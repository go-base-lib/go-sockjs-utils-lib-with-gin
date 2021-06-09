package conn

import "github.com/gorilla/websocket"

type ConnectionBuf struct {
	*websocket.Conn
}

func NewConnectionBuf(wsConn *websocket.Conn) *ConnectionBuf {
	return &ConnectionBuf{
		Conn: wsConn,
	}
}
