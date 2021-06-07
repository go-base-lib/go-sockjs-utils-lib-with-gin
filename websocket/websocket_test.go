package websocket

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestWebSocket(t *testing.T) {

	engine := NewWebSocket("/dev")
	engine.GET("/", func(context *gin.Context) {
		context.String(200, "1sdf")
	})
	if err := engine.Run(":8030"); err != nil {
		panic(err)
	}

	fmt.Println("完成")
}
