package websocket

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewWebSocket(socketUrl string) *Engine {
	engine := gin.Default()
	return NewWebSocketByGin(engine, socketUrl)
}

func NewWebSocketByGin(engine *gin.Engine, socketUrl string) *Engine {
	websocketEngine := &Engine{
		Engine: engine,
	}
	engine.GET(socketUrl, func(context *gin.Context) {
		ws, err := upGrader.Upgrade(context.Writer, context.Request, nil)
		if err != nil {
			context.JSON(500, gin.H{
				"error": true,
				"msg":   "升级协议失败 => " + err.Error(),
				"code":  "1",
			})
			return
		}
		websocketEngine.handleWs(ws)
	})
	return websocketEngine
}
