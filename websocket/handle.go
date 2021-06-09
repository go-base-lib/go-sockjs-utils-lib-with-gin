package websocket

import (
	"bytes"
	"fmt"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/conn"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/socket"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"time"
)

const (
	maxMessageSize = 1024 * 1024
)

type engineHandle struct {
	*Engine
	wsConnBuf     *conn.ConnectionBuf
	connBufReader *bufWebsocketReader
	//wsConnReadBuf *bufio.Reader
	readBufSlice []byte
	readLastBuf  *bytes.Buffer
}

// begin 开始
func (this *engineHandle) begin() {
	defer func() { recover() }()
	defer func() { this.wsConnBuf.Close() }()
	this.readBufSlice = make([]byte, maxMessageSize, maxMessageSize)
	//this.wsConnReadBuf = bufio.NewReader(this.wsConn.UnderlyingConn())
	this.readLastBuf = &bytes.Buffer{}
	this.connBufReader = newBufWebsocketReader(this.wsConnBuf)
	go this.writeLoop()
	this.readLoop()
}

// readLoop 读循环
func (this *engineHandle) readLoop() {

	this.wsConnBuf.SetReadLimit(maxMessageSize)
	_ = this.wsConnBuf.SetReadDeadline(time.Time{})
	this.wsConnBuf.SetPongHandler(func(string) error { _ = this.wsConnBuf.SetReadDeadline(time.Time{}); return nil })
	for {
		context, err := this.readMsgContext()
		if err == io.EOF {
			return
		}

		if err != nil {
			_, ok := err.(*websocket.CloseError)
			if ok || err == net.ErrClosed {
				return
			}

			_, ok = err.(*net.OpError)
			if ok {
				return
			}

			//this.wsConn.write
			fmt.Println("消息块读取失败")
			continue
		}

		handleFn, ok := this.matchCmd(context.Cmd())
		if !ok {
			fmt.Println("404")
			continue
		}
		handleFn(context)
	}
}

func (this *engineHandle) writeLoop() {

}

// readMsgContext 读取一个context消息
func (this *engineHandle) readMsgContext() (*socket.Context, error) {
	//_, r, err := this.wsConn.NextReader()
	//if err != nil {
	//	return nil, err
	//}

	//tmpBuffReader := bufio.NewReader(r)
	cmdUrl, err := this.connBufReader.ReadLine()
	if err != nil {
		return nil, err
	}

	msgId, err := this.connBufReader.ReadLine()
	if err != nil {
		return nil, err
	}

	isFileStr, err := this.connBufReader.ReadLine()
	if err != nil {
		return nil, err
	}

	isFile := true
	if string(isFileStr) == "1" {
		isFile = false
	}

	msgFile := ""
	if isFile {
		filePath, err := this.connBufReader.ReadLine()
		if err != nil {
			return nil, err
		}
		msgFile = filePath
	} else {
		datLenStr, err := this.connBufReader.ReadLine()
		if err != nil {
			return nil, ErrReadMsgContent
		}
		dataLen, err := strconv.ParseInt(string(datLenStr), 10, 64)
		if err != nil {
			return nil, ErrReadMsgContent
		}

		msgFile, err = this.readSizeContentToFile(int(dataLen))
		if err != nil {
			return nil, err
		}
	}

	return socket.NewWebSocketContext(this.wsConnBuf, cmdUrl, msgId, isFileStr, msgFile), nil
}

func (this *engineHandle) readSizeContentToFile(size int) (string, error) {
	tmpFile, err := ioutil.TempFile("devPlatform", "*")
	if err != nil {
		return "", ErrCreateContentFile
	}
	defer tmpFile.Close()

	totalReadSize := this.readLastBuf.Len()
	if totalReadSize >= size {
		lastData := this.readLastBuf.Bytes()
		writeData := lastData[:size]
		lastData = lastData[size:]
		_, err = tmpFile.Write(writeData)
		if err != nil {
			return "", ErrReadMsgContent
		}

		this.readLastBuf.Reset()
		if len(lastData) > 0 {
			this.readLastBuf.Write(lastData)
		}
		return tmpFile.Name(), nil
	} else {
		_, err = tmpFile.Write(this.readLastBuf.Bytes())
		if err != nil {
			return "", ErrReadMsgContent
		}
		this.readLastBuf.Reset()
	}

	for {
		readLen, err := this.connBufReader.Read(this.readBufSlice)
		//if err == io.EOF {
		//	_, r, err := this.wsConn.NextReader()
		//	if err != nil {
		//		return nil, err
		//	}
		//	currentBufferReader = bufio.NewReader(r)
		//	continue
		//}
		//readLen, err = this.wsConnReadBuf.Read(this.readBufSlice)
		//if err != nil {
		//	return nil, ErrReadMsgContent
		//}
		totalReadSize += readLen
		otherSize := totalReadSize - size
		isOk := false
		if totalReadSize >= size {
			readLen -= otherSize
			isOk = true
		}

		readData := this.readBufSlice[:readLen]
		_, err = tmpFile.Write(readData)
		if err != nil {
			return "", ErrReadMsgContent
		}

		if isOk {
			if otherSize > 0 {
				this.readLastBuf.Write(this.readBufSlice[readLen : readLen+otherSize])
			}
			return tmpFile.Name(), nil
		}

	}
}
