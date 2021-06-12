package websocket

import (
	"fmt"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/conn"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"time"
)

const (
	maxMessageSize = 1024 * 1024
)

type HookContext interface {
	SendMsgAndReturnWithTimeout(cmd string, modType conn.ModType, sendData interface{}, timeout time.Duration) (*conn.Context, error)
	SendMsg(cmd string, modType conn.ModType, sendData interface{}) error
}

type engineHandle struct {
	*conn.Context
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
	if hookFn, ok := this.hookMapper[HookNameOpen]; ok {
		go hookFn(this)
	}
	for {
		context, err := this.readMsgContext()
		if err == io.EOF {
			return
		}

		if err != nil {
			_, ok := err.(*websocket.CloseError)
			if ok || err == net.ErrClosed {
				if hookFn, ok := this.hookMapper[HookNameClose]; ok {
					hookFn(this)
					this.Destroy()
				}
				return
			}

			_, ok = err.(*net.OpError)
			if ok {
				return
			}

			fmt.Println("消息块读取失败")
			continue
		}

		go func() {
			defer func() { recover() }()
			defer context.Destroy()
			handleFn, ok := this.matchCmd(context.Cmd())
			if !ok {
				_ = context.ReturnErr("404", "未找到对应请求命令")
				fmt.Println("404")
				return
			}
			err = handleFn(context)
			if err != nil {
				context.ReturnErr("500", err.Error())
				return
			}
			if context.NeedReturn() && !context.IsReturn() {
				_ = context.ReturnVoid()
			}
		}()
	}
}

// readMsgContext 读取一个context消息
func (this *engineHandle) readMsgContext() (*conn.Context, error) {
	info, err := this.wsConnBuf.ReadMsgInfo()
	if err != nil {
		return nil, err
	}
	return conn.NewWebSocketContext(this.wsConnBuf, info.Cmd, info.NeedReturn(), info.MsgId, info.Mod, info.Data), err
}
