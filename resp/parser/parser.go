package parser

import (
	"bufio"
	"errors"
	"goredies/interface/resp"
	"goredies/lib/logger"
	"goredies/resp/reply"
	"io"
	"runtime/debug"
	"strconv"
	"strings"
)

type Payload struct {
	Data resp.Reply
	Err  error
}

// 状态
type readState struct {
	readingMultiLine  bool     //解析多行多行
	expectedArgsCount int      //读取 应该有几个参数
	msgType           byte     //消息类型
	args              [][]byte //数据
	bulkLen           int64    //字节组 数据块长度
}

func (s *readState) finished() bool { //解析出来与目标一样多 解析结束
	return s.expectedArgsCount > 0 && len(s.args) == s.expectedArgsCount
}

// ParseStream 通过 io.Reader 读取数据并将结果通过 channel 将结果返回给调用者
// 流式处理的接口适合供客户端/服务端使用
func ParseStream(reader io.Reader) <-chan *Payload {
	ch := make(chan *Payload)
	go parse0(reader, ch)
	return ch
}

// 解析
func parse0(reader io.Reader, ch chan<- *Payload) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(string(debug.Stack()))
		}
	}()
	bufReader := bufio.NewReader(reader)
	var state readState
	var err error
	var msg []byte
	for {
		var ioErr bool
		msg, ioErr, err = readLine(bufReader, &state) //读一行
		if err != nil {
			if ioErr { //是不是io
				ch <- &Payload{
					Err: err,
				}
				close(ch)
				return
			}
			ch <- &Payload{
				Err: err,
			}
			state = readState{}
			continue
		}

		// 单行: StatusReply, IntReply, ErrorReply
		// 多行: BulkReply, MultiBulkReply
		// 判断是不是多行解析模式
		if !state.readingMultiLine {
			if msg[0] == '*' { // *3\r\n
				err = parseMultiBulkHeader(msg, &state) //提取个数
				if err != nil {
					ch <- &Payload{
						Err: errors.New("protocol error:" + string(msg)),
					}
					state = readState{}
					continue
				}
				if state.expectedArgsCount == 0 {
					ch <- &Payload{
						Data: &reply.EmptyMultiBulkReply{},
					}
					state = readState{}
					continue
				}
			} else if msg[0] == '$' { // $4\r\nPING\r\n or $-1\r\n
				err = parseBulkHeader(msg, &state)
				if err != nil {
					ch <- &Payload{
						Err: errors.New("protocol error:" + string(msg)),
					}
					state = readState{}
					continue
				}
				if state.bulkLen == -1 {
					ch <- &Payload{
						Data: &reply.EmptyMultiBulkReply{},
					}
					state = readState{}
					continue
				}
			} else {
				result, err := parseSingleLineReply(msg)
				ch <- &Payload{
					Data: result,
					Err:  err,
				}
				state = readState{}
				continue
			}
		} else {
			err = readBody(msg, &state)
			if err != nil {
				ch <- &Payload{
					Err: errors.New("protocol error:" + string(msg)),
				}
				state = readState{}
				continue
			}
			if state.finished() {
				var result resp.Reply
				if state.msgType == '*' {
					result = reply.MakeMultiBulkReply(state.args)
				} else if state.msgType == '$' {
					result = reply.MakeBulkReply(state.args[0])
				}
				ch <- &Payload{
					Data: result,
					Err:  err,
				}
				state = readState{}
			}
		}
	}
}

// 取出一行 遇到$7\r\n k\r\ney\r\n 要切换读法
// *3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
func readLine(bufReader *bufio.Reader, state *readState) ([]byte, bool, error) {
	var msg []byte
	var err error
	if state.bulkLen == 0 { //1.\r\n切分
		msg, err = bufReader.ReadBytes('\n')
		if err != nil {
			return nil, true, err
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' {
			return nil, false, errors.New("protocol error:" + string(msg))
		}
	} else { //2.之前读到了$数字，严格读取字符个数
		msg = make([]byte, state.bulkLen+2)
		_, err = io.ReadFull(bufReader, msg)
		if err != nil {
			return nil, true, err
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' || msg[len(msg)-1] != '\n' {
			return nil, false, errors.New("protocol error:" + string(msg))
		}
		state.bulkLen = 0
	}
	return msg, false, nil
}

// 解析数组
// *3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
func parseMultiBulkHeader(msg []byte, state *readState) error {
	var err error
	var expectedLine uint64 //读到* 要读多少行
	expectedLine, err = strconv.ParseUint(string(msg[1:len(msg)-2]), 10, 32)
	if err != nil {
		return errors.New("protocol error:" + string(msg))
	}
	if expectedLine == 0 {
		state.expectedArgsCount = 0
		return nil
	} else if expectedLine > 0 {
		state.msgType = msg[0] //给* 表示读数组
		state.readingMultiLine = true
		state.expectedArgsCount = int(expectedLine)
		state.args = make([][]byte, 0, expectedLine)
		return nil
	} else {
		return errors.New("protocol error:" + string(msg))
	}
}

// 单行 $
// $4\r\nPING\r\n
func parseBulkHeader(msg []byte, state *readState) error {
	var err error
	state.bulkLen, err = strconv.ParseInt(string(msg[1:len(msg)-2]), 10, 64)
	if err != nil {
		return errors.New("protocol error:" + string(msg))
	}
	if state.bulkLen == -1 {
		return nil
	} else if state.bulkLen > 0 {
		state.msgType = msg[0]
		state.readingMultiLine = true
		state.expectedArgsCount = 1
		state.args = make([][]byte, 0, 1)
		return nil
	} else {
		return errors.New("protocol error:" + string(msg))
	}
}

// +OK\r\n -err\r\n :5\r\n
func parseSingleLineReply(msg []byte) (resp.Reply, error) {
	str := strings.TrimSuffix(string(msg), "\r\n") //切 去掉\r\n
	var result resp.Reply
	switch msg[0] {
	case '+':
		result = reply.MakeStatusReply(str[1:])
	case '-':
		result = reply.MakeErrReply(str[1:])
	case ':':
		val, err := strconv.ParseInt(str[1:], 10, 64)
		if err != nil {
			return nil, errors.New("protocol error:" + string(msg))
		}
		result = reply.MakeIntReply(val)
	}
	return result, nil
}

// $3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n 剩数组
// PING\r\n 剩内容
func readBody(msg []byte, state *readState) error {
	line := msg[0 : len(msg)-2]
	var err error
	//$3
	if line[0] == '$' {
		state.bulkLen, err = strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			return errors.New("protocol error:" + string(msg))
		}
		//$0\r\n
		if state.bulkLen <= 0 {
			state.args = append(state.args, []byte{})
			state.bulkLen = 0
		}
	} else {
		state.args = append(state.args, line)
	}
	return nil
}
