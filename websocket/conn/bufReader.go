package conn

import (
	"bufio"
	"bytes"
	"github.com/gorilla/websocket"
	"io"
)

type bufWebsocketReader struct {
	conn      *websocket.Conn
	bufReader *bufio.Reader
	tmpMsg    *bytes.Buffer
}

func (this *bufWebsocketReader) initReader() error {
	if this.bufReader == nil {
		_, r, err := this.conn.NextReader()
		if err != nil {
			return err
		}
		this.bufReader = bufio.NewReader(r)
	}
	return nil
}

func (this *bufWebsocketReader) ReadLine() (string, error) {
ConnectHeader:
	if err := this.initReader(); err != nil {
		return "", err
	}
	line, err := this.bufReader.ReadString('\n')
	if err == io.EOF {
		this.tmpMsg.WriteString(line)
		this.bufReader = nil
		goto ConnectHeader
	}
	if err != nil {
		return "", err
	}
	if this.tmpMsg.Len() > 0 {
		line = this.tmpMsg.String() + line
		this.tmpMsg.Reset()
	}
	if len(line) < 1 {
		goto ConnectHeader
	}
	line = line[:len(line)-1]

	return line, nil
}

func (this *bufWebsocketReader) Read(bytes []byte) (int, error) {
ConnectHeader:
	if err := this.initReader(); err != nil {
		return 0, err
	}

	readLen, err := this.bufReader.Read(bytes)
	if err == io.EOF {
		this.bufReader = nil
		goto ConnectHeader
	}
	if err != nil {
		return 0, err
	}
	return readLen, nil
}

func newBufWebsocketReader(conn *websocket.Conn) *bufWebsocketReader {
	return &bufWebsocketReader{
		conn:   conn,
		tmpMsg: &bytes.Buffer{},
	}
}
