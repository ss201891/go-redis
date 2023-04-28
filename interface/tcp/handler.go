package tcp

import (
	"context"
	"net"
)

type Handler interface {
	//传递一些参数，如超时，环境参数
	Handle(ctx context.Context, conn net.Conn)
	Close() error
}
