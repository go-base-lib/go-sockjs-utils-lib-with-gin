package sockjs

import (
	"github.com/devloperPlatform/go-sockjs-utils-lib-with-gin/sockjs/conn"
	"github.com/gin-gonic/gin"
	"github.com/igm/sockjs-go/v3/sockjs"
)

type HandleFn func(ctx *conn.Context) error
type HookFn func(engine HookContext)
type MiddlewareFn func(ctx *conn.Context) error

type HookName string

const (
	HookNameOpen  HookName = "open"
	HookNameClose HookName = "close"
)

type Engine struct {
	*gin.Engine
	handleMapper   map[string]HandleFn
	hookMapper     map[HookName]HookFn
	middlewareList []MiddlewareFn
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

func (this *Engine) Middleware(middlewareFn MiddlewareFn) {
	if this.middlewareList == nil {
		this.middlewareList = make([]MiddlewareFn, 8)
	}
	this.middlewareList = append(this.middlewareList, middlewareFn)
}

func (this *Engine) handleWs(wsConn *sockjs.Session) {
	handle := &engineHandle{
		Engine:    this,
		wsConnBuf: conn.NewConnectionBuf(wsConn),
	}
	handle.Context = conn.NewWebSocketContext(handle.wsConnBuf, "", false, "", "", "")
	go handle.begin()
}

func (this *Engine) matchCmd(cmdStr string) (HandleFn, bool) {
	fn, ok := this.handleMapper[cmdStr]
	return fn, ok
}
