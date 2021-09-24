package sockjs

import (
	"github.com/gin-gonic/gin"
	"github.com/igm/sockjs-go/v3/sockjs"
	"strings"
)

func NewWebSocketServer(socketUrl string, options *sockjs.Options) *Engine {
	engine := gin.Default()
	return NewWebSocketByGin(engine, socketUrl, options)
}

func NewWebSocketByGin(engine *gin.Engine, socketUrl string, options *sockjs.Options) *Engine {
	websocketEngine := &Engine{
		Engine: engine,
	}

	if options == nil {
		options = &sockjs.DefaultOptions
	}

	sockjsHandler := sockjs.NewHandler(socketUrl, *options, func(session sockjs.Session) {
		websocketEngine.handleWs(&session)
	})
	if !strings.HasSuffix(socketUrl, "/") {
		socketUrl += "/"
	}
	engine.Any(socketUrl+"*path", gin.WrapH(sockjsHandler))
	//engine.GET(socketUrl, func(context *gin.Context) {
	//	ws, err := upGrader.Upgrade(context.Writer, context.Request, nil)
	//	if err != nil {
	//		context.JSON(500, gin.H{
	//			"error": true,
	//			"msg":   "升级协议失败 => " + err.Error(),
	//			"code":  "1",
	//		})
	//		return
	//	}
	//	websocketEngine.handleWs(ws)
	//})
	return websocketEngine
}
