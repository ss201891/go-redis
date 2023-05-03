package reply

type PongReply struct {
}

var pingbytes = []byte("+PONG\r\n")

func (r PongReply) ToBytes() []byte {
	return pingbytes
}

func MakePongReply() *PongReply {
	return &PongReply{}
}

type OkReply struct{}

var okBytes = []byte("+OK\r\n")

func (r *OkReply) ToBytes() []byte {
	return okBytes
}

var theOkReply = new(OkReply)

func MakeOkReply() *OkReply {
	return theOkReply
}

type NullBulkReply struct {
}

var nullBulkBytes = []byte("$-1\r\n")

func (r NullBulkReply) ToBytes() []byte {
	return nullBulkBytes
}

func MakeNullBulkReply() *NullBulkReply {
	return &NullBulkReply{}
}

type EmptyMultiBulkReply struct{}

var emptyMultiBulkBytes = []byte("*0\r\n")

func (r *EmptyMultiBulkReply) ToBytes() []byte {
	return emptyMultiBulkBytes
}

type NoReply struct{}

var noBytes = []byte("")

func (n *NoReply) ToBytes() []byte {
	return noBytes
}





