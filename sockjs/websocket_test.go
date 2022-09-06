package sockjs

import (
	"fmt"
	"github.com/go-base-lib/go-sockjs-utils-lib-with-gin/sockjs/conn"
	"github.com/igm/sockjs-go/v3/sockjs"
	"testing"
)

func TestWebSocket(t *testing.T) {

	engine := NewWebSocketServer("/dev", &sockjs.DefaultOptions)
	engine.Handle("hello", func(ctx *conn.Context) error {
		type tmpStrut struct {
			Hello string `json:"hello"`
		}
		tmpData := &tmpStrut{}
		//tmpData := ""
		_ = ctx.Unmarshal(&tmpData)
		fmt.Println(tmpData)
		andReturn, err := ctx.SendMsgAndReturn("/hello", conn.ModTypeMem, "asdadsf")
		if err != nil {
			return err
		}

		tmpStr := ""
		err = andReturn.Unmarshal(&tmpStr)
		if err != nil {
			return err
		}
		fmt.Println(tmpStr)
		//ctx.ReturnData(2)
		return nil
	})
	////engine.GET("/", func(context *gin.Context) {
	////	context.String(200, "1sdf")
	////})
	if err := engine.Run(":65521"); err != nil {
		panic(err)
	}

	fmt.Println("完成")
}
