package websocket

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type HandleFn func(ctx *Context)

type Engine struct {
	*gin.Engine
	handleMapper map[string]HandleFn
}

// Handle 拦截命令
func (this *Engine) Handle(cmdStr string, handleFn HandleFn) *Engine {
	this.handleMapper[cmdStr] = handleFn
	return this
}

func (this *Engine) handleWs(wsConn *websocket.Conn) {
	handle := &engineHandle{
		Engine: this,
		wsConn: wsConn,
	}

	go handle.begin()
}

func (this *Engine) matchCmd(cmdStr string) (HandleFn, bool) {
	fn, ok := this.handleMapper[cmdStr]
	return fn, ok
}
