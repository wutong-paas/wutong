package event

import (
	"os"
	"sync"
	"time"

	"github.com/wutong-paas/wutong/pkg/gogo"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/discover/config"
	eventclient "github.com/wutong-paas/wutong/eventlog/entry/grpc/client"
	eventpb "github.com/wutong-paas/wutong/eventlog/entry/grpc/pb"
	"github.com/wutong-paas/wutong/util"
	"golang.org/x/net/context"
)

var eventLogServer = "wt-eventlog:6366"

func init() {
	if s := os.Getenv("EVENT_LOG_SERVER"); s != "" {
		eventLogServer = s
	}
}

// GetLogger 获取日志，使用完成后必须调用 CloseLogger 方法
func GetLogger(eventID string) Logger {
	return defaultLoggerManager.GetLogger(eventID)
}

// CloseLogger 关闭日志
func CloseLogger(eventID string) {
	if defaultLoggerManager != nil {
		logger := defaultLoggerManager.GetLogger(eventID)
		if logger != nil {
			defaultLoggerManager.ReleaseLogger(logger)
		}
	}
}

// LoggerManager 操作日志，客户端服务
// 客户端负载均衡
type LoggerManager interface {
	GetLogger(eventID string) Logger
	Start() error
	Close()
	ReleaseLogger(Logger)
}

type loggerManager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	loggers map[string]Logger
	handle  *handle
	lock    sync.Mutex
}

var defaultLoggerManager LoggerManager

const buffersize = 1000

// NewLoggerManager 创建 loggerManager
func NewLoggerManager() (LoggerManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defaultLoggerManager = &loggerManager{
		ctx:     ctx,
		cancel:  cancel,
		loggers: make(map[string]Logger, 1024),
	}
	err := defaultLoggerManager.Start()

	return defaultLoggerManager, err
}

// Start 开始日志服务
func (m *loggerManager) Start() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	gogo.Go(func(ctx context.Context) error {
		for {
			h := &handle{
				cacheChan: make(chan []byte, buffersize),
				manager:   m,
				ctx:       m.ctx,
			}
			m.handle = h
			err := h.HandleLog()
			if err != nil {
				time.Sleep(time.Second * 10)
				logrus.Warnf("event log server connect error: %v. auto retry after 10 seconds ", err)
				continue
			}
			return nil
		}
	})

	go m.GC()
	return nil
}

// UpdateEndpoints - 不需要去更新节点信息
func (m *loggerManager) UpdateEndpoints(endpoints ...*config.Endpoint) {}

// Error 异常信息
func (m *loggerManager) Error(err error) {}

// Close 关闭日志服务
func (m *loggerManager) Close() {
	if m != nil {
		logrus.Warn("event log manager ctx cancled.")
		m.cancel()
	}
}

// GC 主动调用，回收资源
func (m *loggerManager) GC() {
	util.IntermittentExec(m.ctx, func() {
		m.lock.Lock()
		defer m.lock.Unlock()
		var needRelease []string
		for k, l := range m.loggers {
			//1 min 未 release ,自动 GC
			if l.CreateTime().Add(time.Minute).Before(time.Now()) {
				needRelease = append(needRelease, k)
			}
		}
		if len(needRelease) > 0 {
			for _, event := range needRelease {
				logrus.Infof("start auto release event logger. %s", event)
				delete(m.loggers, event)
			}
		}
	}, time.Second*20)
}

// GetLogger 使用完成后必须调用 ReleaseLogger 方法
func (m *loggerManager) GetLogger(eventID string) Logger {
	m.lock.Lock()
	defer m.lock.Unlock()
	if eventID == " " || len(eventID) == 0 {
		eventID = "system"
	}
	if l, ok := m.loggers[eventID]; ok {
		return l
	}
	l := NewLogger(eventID, m.getCacheChan())
	m.loggers[eventID] = l
	return l
}

// ReleaseLogger 释放 logger
func (m *loggerManager) ReleaseLogger(l Logger) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if l, ok := m.loggers[l.Event()]; ok {
		delete(m.loggers, l.Event())
	}
}

// SetNewHandleCacheChan 设置新的 handle cache chan
func (m *loggerManager) SetNewHandleCacheChan(cacheChan chan []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.handle != nil && m.handle.cacheChan == cacheChan {
		logrus.Warnf("event log server can not link, will ignore it.")
	}

	for _, v := range m.loggers {
		if v.GetChan() == cacheChan {
			v.SetChan(m.getCacheChan())
		}
	}
}

type handle struct {
	cacheChan chan []byte
	ctx       context.Context
	manager   *loggerManager
}

func (m *loggerManager) getCacheChan() chan []byte {
	if m.handle != nil {
		return m.handle.cacheChan
	}

	m.handle = &handle{
		cacheChan: make(chan []byte, buffersize),
		manager:   m,
		ctx:       m.ctx,
	}
	go m.handle.HandleLog()
	return m.handle.cacheChan
}

// RemoveHandle 移除当前 handle
func (m *loggerManager) RemoveHandle() {
	m.lock.Lock()
	defer m.lock.Unlock()
	// delete if server exist
	m.handle = nil
}

// HandleLog -
func (h *handle) HandleLog() error {
	defer h.manager.RemoveHandle()
	return util.Exec(h.ctx, func() error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		client, err := eventclient.NewEventClient(ctx, eventLogServer)
		if err != nil {
			logrus.Error("create event client error: ", err.Error())
			return err
		}
		logrus.Infof("start a event log handle core.")
		logClient, err := client.Log(ctx)
		if err != nil {
			logrus.Error("create event log client error: ", err.Error())
			// 切换使用此 chan 的 logger 到其他 chan
			h.manager.SetNewHandleCacheChan(h.cacheChan)
			return err
		}
		for {
			select {
			case <-h.ctx.Done():
				logrus.Warn("h ctx done")
				logClient.CloseSend()
				return nil
			case me := <-h.cacheChan:
				err := logClient.Send(&eventpb.LogMessage{Log: me})
				if err != nil {
					logrus.Error("send event log error: ", err.Error())
					logClient.CloseSend()
					// 切换使用此 chan 的 logger 到其他 chan
					h.manager.SetNewHandleCacheChan(h.cacheChan)
					return nil
				}
			}
		}
	}, time.Second*3)
}
