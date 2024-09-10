package component

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/controller"
	"github.com/wutong-paas/wutong/api/db"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/server"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/pkg/component/cr"
	"github.com/wutong-paas/wutong/pkg/component/etcd"
	"github.com/wutong-paas/wutong/pkg/component/grpc"
	"github.com/wutong-paas/wutong/pkg/component/k8s"
	"github.com/wutong-paas/wutong/pkg/component/mq"
	"github.com/wutong-paas/wutong/pkg/component/prom"
	"github.com/wutong-paas/wutong/pkg/wutong"
)

// Database -
func Database() wutong.Component {
	return db.Database()
}

// K8sClient -
func K8sClient() wutong.Component {
	return k8s.Client()
}

// HubRegistry -
func HubRegistry() wutong.Component {
	return cr.HubRegistry()
}

// Etcd -
func Etcd() wutong.Component {
	return etcd.Etcd()
}

// MQ -
func MQ() wutong.Component {
	return mq.MQ()
}

// Prometheus -
func Prometheus() wutong.Component {
	return prom.Prometheus()
}

// Grpc -
func Grpc() wutong.Component {
	return grpc.Grpc()
}

// Event -
func Event() wutong.FuncComponent {
	logrus.Infof("init event...")
	return func(ctx context.Context, cfg *configs.Config) error {
		var tryTime time.Duration
		var err error
		for tryTime < 4 {
			tryTime++
			if _, err = event.NewLoggerManager(); err != nil {
				logrus.Errorf("get event manager failed, try time is %v,%s", tryTime, err.Error())
				time.Sleep((5 + tryTime*10) * time.Second)
			} else {
				break
			}
		}
		if err != nil {
			logrus.Errorf("get event manager failed. %v", err.Error())
			return err
		}
		logrus.Info("init event manager success")
		return nil
	}
}

// Handler -
func Handler() wutong.FuncComponent {
	return func(ctx context.Context, cfg *configs.Config) error {
		return handler.InitHandle(cfg.APIConfig)
	}
}

// Router -
func Router() wutong.FuncComponent {
	return func(ctx context.Context, cfg *configs.Config) error {
		if err := controller.CreateV2RouterManager(cfg.APIConfig, grpc.Default().StatusClient); err != nil {
			logrus.Errorf("create v2 route manager error, %v", err)
		}
		// 启动api
		apiManager := server.NewManager(cfg.APIConfig, etcd.Default().EtcdClient)
		if err := apiManager.Start(); err != nil {
			return err
		}
		logrus.Info("api router is running...")
		return nil
	}
}

// Proxy -
func Proxy() wutong.FuncComponent {
	return func(ctx context.Context, cfg *configs.Config) error {
		handler.InitProxy(cfg.APIConfig)
		return nil
	}
}
