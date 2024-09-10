package event

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/wutong-paas/wutong/util"
)

// Logger 日志发送器
type Logger interface {
	Info(string, map[string]string)
	Error(string, map[string]string)
	Debug(string, map[string]string)
	Event() string
	CreateTime() time.Time
	GetChan() chan []byte
	SetChan(chan []byte)
	GetWriter(step, level string) LoggerWriter
}

// NewLogger creates a new Logger.
func NewLogger(eventID string, sendCh chan []byte) Logger {
	return &logger{
		event:      eventID,
		sendChan:   sendCh,
		createTime: time.Now(),
	}
}

type logger struct {
	event      string
	sendChan   chan []byte
	createTime time.Time
}

// GetChan -
func (l *logger) GetChan() chan []byte {
	return l.sendChan
}

// SetChan -
func (l *logger) SetChan(ch chan []byte) {
	l.sendChan = ch
}
func (l *logger) Event() string {
	return l.event
}
func (l *logger) CreateTime() time.Time {
	return l.createTime
}
func (l *logger) Info(message string, info map[string]string) {
	if info == nil {
		info = make(map[string]string)
	}
	info["level"] = "info"
	l.send(message, info)
}
func (l *logger) Error(message string, info map[string]string) {
	if info == nil {
		info = make(map[string]string)
	}
	info["level"] = "error"
	l.send(message, info)
}
func (l *logger) Debug(message string, info map[string]string) {
	if info == nil {
		info = make(map[string]string)
	}
	info["level"] = "debug"
	l.send(message, info)
}

func (l *logger) send(message string, info map[string]string) {
	info["event_id"] = l.event
	info["message"] = message
	info["time"] = time.Now().Format(time.RFC3339)
	log, err := ffjson.Marshal(info)
	if err == nil && l.sendChan != nil {
		util.SendNoBlocking(log, l.sendChan)
	}
}

// LoggerWriter logger writer
type LoggerWriter interface {
	io.Writer
	SetFormat(map[string]interface{})
}

func (l *logger) GetWriter(step, level string) LoggerWriter {
	return &loggerWriter{
		logger: l,
		step:   step,
		level:  level,
	}
}

type loggerWriter struct {
	logger      *logger
	step        string
	level       string
	fmt         map[string]interface{}
	tmp         []byte
	lastMessage string
}

func (w *loggerWriter) SetFormat(f map[string]interface{}) {
	w.fmt = f
}
func (w *loggerWriter) Write(b []byte) (n int, err error) {
	if len(b) > 0 {
		if !strings.HasSuffix(string(b), "\n") {
			w.tmp = append(w.tmp, b...)
			return len(b), nil
		}
		var message string
		if len(w.tmp) > 0 {
			message = string(append(w.tmp, b...))
			w.tmp = w.tmp[:0]
		} else {
			message = string(b)
		}

		// if loggerWriter has format, and then use it format message
		if len(w.fmt) > 0 {
			newLineMap := make(map[string]interface{}, len(w.fmt))
			for k, v := range w.fmt {
				if v == "%s" {
					newLineMap[k] = fmt.Sprintf(v.(string), message)
				} else {
					newLineMap[k] = v
				}
			}
			messageb, _ := ffjson.Marshal(newLineMap)
			message = string(messageb)
		}
		if w.step == "build-progress" {
			if strings.HasPrefix(message, "Progress ") && strings.HasPrefix(w.lastMessage, "Progress ") {
				w.lastMessage = message
				return len(b), nil
			}
			// send last message
			if !strings.HasPrefix(message, "Progress ") && strings.HasPrefix(w.lastMessage, "Progress ") {
				w.logger.send(message, map[string]string{"step": w.lastMessage, "level": w.level})
			}
		}
		w.logger.send(message, map[string]string{"step": w.step, "level": w.level})
		w.lastMessage = message
	}
	return len(b), nil
}
