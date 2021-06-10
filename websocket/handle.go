package websocket

import (
	"fmt"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/conn"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/data"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"time"
)

const (
	maxMessageSize = 1024 * 1024
)

type engineHandle struct {
	*Engine
	wsConnBuf *conn.ConnectionBuf
	//connBufReader *bufWebsocketReader
	//wsConnReadBuf *bufio.Reader
	//readBufSlice []byte
	//readLastBuf  *bytes.Buffer
}

// begin 开始
func (this *engineHandle) begin() {
	defer func() { recover() }()
	defer func() { this.wsConnBuf.Close() }()
	this.readLoop()
}

// readLoop 读循环
func (this *engineHandle) readLoop() {

	this.wsConnBuf.SetReadLimit(maxMessageSize)
	_ = this.wsConnBuf.SetReadDeadline(time.Time{})
	this.wsConnBuf.SetPongHandler(func(string) error { _ = this.wsConnBuf.SetReadDeadline(time.Time{}); return nil })
	for {
		context, err := this.readMsgContext()
		if err == io.EOF {
			return
		}

		if err != nil {
			_, ok := err.(*websocket.CloseError)
			if ok || err == net.ErrClosed {
				return
			}

			_, ok = err.(*net.OpError)
			if ok {
				return
			}

			fmt.Println("消息块读取失败")
			continue
		}

		handleFn, ok := this.matchCmd(context.Cmd())
		if !ok {
			errDataStr, _ := data.MarshalErr("404", "未找到对应请求命令")
			_ = this.wsConnBuf.SendMsg(&conn.MsgInfo{
				Mod:   context.Mod(),
				MsgId: context.MsgId(),
				Data:  errDataStr,
			})
			fmt.Println("404")
			continue
		}
		handleFn(context)
	}
}

// readMsgContext 读取一个context消息
func (this *engineHandle) readMsgContext() (*conn.Context, error) {
	info, err := this.wsConnBuf.ReadMsgInfo()
	if err != nil {
		return nil, err
	}
	return conn.NewWebSocketContext(this.wsConnBuf, info.Cmd, info.MsgId, info.Mod, info.Data), err
}
