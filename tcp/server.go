package tcp

import (
	"context"
	"goredies/interface/tcp"
	"goredies/lib/logger"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Config struct {
	Address string
}

// 监听然后服务信号
func ListenAndServeWithSignal(cfg *Config, handler tcp.Handler) error {
	closeChan := make(chan struct{})
	signChan := make(chan os.Signal)
	signal.Notify(signChan, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT) //系统信号转发给singnchan,挂起杀掉
	go func() {
		sign := <-signChan
		switch sign {
		case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			closeChan <- struct{}{}

		}
	}()
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}
	logger.Info("start Listen")
	ListenAndServe(listener, handler, closeChan)
	return nil
}

func ListenAndServe(listener net.Listener, handler tcp.Handler, closeChan <-chan struct{}) {
	go func() {
		<-closeChan
		logger.Info("shutting down ")
		_ = listener.Close()
		_ = handler.Close()
	}()
	defer func() {
		_ = listener.Close()
		_ = handler.Close()
	}()
	ctx := context.Background()
	var waitDone sync.WaitGroup
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		logger.Info("accepted link")
		waitDone.Add(1)
		go func() {
			defer waitDone.Done()
			handler.Handle(ctx, conn)
		}()

	}
	waitDone.Wait()
}
