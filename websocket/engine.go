package websocket

import (
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/conn"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type HandleFn func(ctx *conn.Context) error
type HookFn func(engine *engineHandle)

type HookName string

const (
	HookNameOpen  HookName = "open"
	HookNameClose HookName = "close"
)

type Engine struct {
	*gin.Engine
	handleMapper map[string]HandleFn
	hookMapper   map[HookName]HookFn
}

func (this *Engine) Hook(name HookName, fn HookFn) {
	if this.hookMapper == nil {
		this.hookMapper = make(map[HookName]HookFn, 2)
	}
	this.hookMapper[name] = fn
}

// Handle 拦截命令
func (this *Engine) Handle(cmdStr string, handleFn HandleFn) *Engine {
	if this.handleMapper == nil {
		this.handleMapper = make(map[string]HandleFn)
	}
	this.handleMapper[cmdStr] = handleFn
	return this
}

func (this *Engine) handleWs(wsConn *websocket.Conn) {
	handle := &engineHandle{
		Engine:    this,
		wsConnBuf: conn.NewConnectionBuf(wsConn),
	}
	handle.Context = conn.NewWebSocketContext(handle.wsConnBuf, "", false, "", "", "")
	if hookFn, ok := this.hookMapper[HookNameOpen]; ok {
		hookFn(handle)
	}
	go handle.begin()
}

func (this *Engine) matchCmd(cmdStr string) (HandleFn, bool) {
	fn, ok := this.handleMapper[cmdStr]
	return fn, ok
}
