package sockjs

import (
	"github.com/devloperPlatform/go-sockjs-utils-lib-with-gin/sockjs/conn"
	"github.com/devloperPlatform/go-sockjs-utils-lib-with-gin/sockjs/logs"
	"io"
	"io/ioutil"
	"time"
)

const (
	maxMessageSize = 1024 * 1024
)

type HookContext interface {
	SendMsgAndReturn(cmd string, modType conn.ModType, sendData interface{}) (*conn.Context, error)
	SendMsgAndReturnWithTimeout(cmd string, modType conn.ModType, sendData interface{}, timeout time.Duration) (*conn.Context, error)
	SendMsg(cmd string, modType conn.ModType, sendData interface{}) error
	SendVoidMsg(cmd string) error
	SendVoidMsgAndReturn(cmd string) (*conn.Context, error)
	SendVoidMsgAndReturnWithTimeout(cmd string, timeout time.Duration) (*conn.Context, error)
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

	logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
		log.DebugF("检测[%s]钩子\n", HookNameOpen)
	})
	if hookFn, ok := this.hookMapper[HookNameOpen]; ok {
		logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
			log.DebugF("发现[%s]钩子即将异步执行\n", HookNameOpen)
		})
		go hookFn(this)
	}
	for {
		context, err := this.readMsgContext()
		if err == io.EOF {
			logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
				log.Debugln("消息读到尾部，退出消息监听")
			})
			return
		}

		logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
			if context == nil {
				return
			}
			file, err := ioutil.ReadFile(context.MsgFilePath())
			if err != nil {
				log.
					DebugF("读取到一条消息, 命令码: [%s], 消息ID: [%s], 传输模式: [%s], 是否需要返回: [%s], 消息内容: [读取失败]",
						context.Cmd(), context.MsgId(), context.Mod(), context.IsReturn())
			} else {
				log.
					DebugF("读取到一条消息, 命令码: [%s], 消息ID: [%s], 传输模式: [%s], 是否需要返回: [%s], 消息内容: [%s]",
						context.Cmd(), context.MsgId(), context.Mod(), context.IsReturn(), string(file))
			}
		})

		if err != nil {
			//_, ok := err.(*websocket.CloseError)
			//if ok || err == net.ErrClosed {
			logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
				log.DebugF("正在检测[%s]钩子", HookNameClose)
			})
			if hookFn, ok := this.hookMapper[HookNameClose]; ok {
				logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
					log.DebugF("检测到钩子[%s], 将被执行")
				})
				hookFn(this)
				this.Destroy()
			}
			return
			//}

			//_, ok = err.(*net.OpError)
			//if ok {
			//	return
			//}

			//continue
		}

		go func() {
			defer func() { recover() }()
			defer context.Destroy()
			handleFn, ok := this.matchCmd(context.Cmd())
			if !ok {
				logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
					log.DebugF("命令[%s]未找到对应的实现方法, 返回404", context.Cmd())
				})
				_ = context.ReturnErr("404", "未找到对应请求命令")
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
