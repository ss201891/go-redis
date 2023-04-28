package tcp

import (
	"bufio"
	"context"
	"goredies/lib/logger"
	"goredies/lib/sync/atomic"
	"goredies/lib/sync/wait"

	"io"
	"net"
	"sync"
	"time"
)

type EchoClient struct {
	Conn    net.Conn
	Waiting wait.Wait //自己实现一个超时功能的wait
}

func (e EchoClient) Close() error {

	// 等待数据发送完成或超时
	e.Waiting.WaitWithTimeout(10 * time.Second)
	_ = e.Conn.Close()
	return nil
}

type EchoHandler struct {
	activeConn sync.Map
	closing    atomic.Boolean //是不是正在关闭
}

func MakeHandler() *EchoHandler {
	return &EchoHandler{}
}

func (handler EchoHandler) Handle(ctx context.Context, conn net.Conn) {
	// 关闭中的handler不会处理新连接
	if handler.closing.Get() {
		_ = conn.Close()
	}
	client := &EchoClient{
		Conn: conn,
	}
	handler.activeConn.Store(client, struct{}{}) // 记住仍然存活的连接

	// 使用 bufio 标准库提供的缓冲区功能
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				logger.Info("Connection close")
				handler.activeConn.Delete(client)
			} else {
				logger.Warn(err)
			}
			return
		}
		client.Waiting.Add(1) //告诉正在工作
		b := []byte(msg)
		_, _ = conn.Write(b)
		client.Waiting.Done()
	}
}

func (handler EchoHandler) Close() error {
	logger.Info("handler shutting down")
	handler.closing.Set(true)

	// 逐个关闭连接
	handler.activeConn.Range(func(key, value interface{}) bool {
		client := key.(*EchoClient)
		_ = client.Conn.Close()
		return true
	})
	return nil
}
