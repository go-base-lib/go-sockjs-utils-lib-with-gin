package sockjs

import (
	"github.com/gin-gonic/gin"
	"github.com/igm/sockjs-go/v3/sockjs"
	"strings"
)


func NewWebSocketServer(socketUrl string) *Engine {
	engine := gin.Default()
	return NewWebSocketByGin(engine, socketUrl)
}

func NewWebSocketByGin(engine *gin.Engine, socketUrl string) *Engine {
	websocketEngine := &Engine{
		Engine: engine,
	}
	sockjsHandler := sockjs.NewHandler(socketUrl, sockjs.DefaultOptions, func(session sockjs.Session) {
		websocketEngine.handleWs(&session)
	})
	if !strings.HasSuffix(socketUrl, "/") {
		socketUrl += "/"
	}
	engine.GET(socketUrl+"*path", gin.WrapH(sockjsHandler))
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
