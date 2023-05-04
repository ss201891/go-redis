package database

import (
	"goredies/interface/resp"
	"goredies/resp/reply"
)

type EchoDatabase struct {
}

func NewEchoDatabase() *EchoDatabase {
	return &EchoDatabase{}
}
func (e EchoDatabase) Exec(client resp.Connection, args [][]byte) resp.Reply {
	return reply.MakeMultiBulkReply(args)
}

func (e EchoDatabase) Close() {
	//TODO implement me
	panic("implement me")
}

func (e EchoDatabase) AfterClientClose(c resp.Connection) {
	//TODO implement me
	panic("implement me")
}
