package websocket

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	maxMessageSize = 1024 * 1024
)

type engineHandle struct {
	*Engine
	wsConn *websocket.Conn
	//wsConnReadBuf *bufio.Reader
	readBufSlice []byte
	readLastBuf  *bytes.Buffer
}

// begin 开始
func (this *engineHandle) begin() {
	defer func() { recover() }()
	defer func() { this.wsConn.Close() }()
	this.readBufSlice = make([]byte, maxMessageSize, maxMessageSize)
	//this.wsConnReadBuf = bufio.NewReader(this.wsConn.UnderlyingConn())
	this.readLastBuf = &bytes.Buffer{}
	go this.writeLoop()
	this.readLoop()
}

// readLoop 读循环
func (this *engineHandle) readLoop() {

	this.wsConn.SetReadLimit(maxMessageSize)
	_ = this.wsConn.SetReadDeadline(time.Time{})
	this.wsConn.SetPongHandler(func(string) error { _ = this.wsConn.SetReadDeadline(time.Time{}); return nil })
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

		handleFn, ok := this.matchCmd(context.Cmd)
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
func (this *engineHandle) readMsgContext() (*Context, error) {
	websocketContext := &Context{}
	_, r, err := this.wsConn.NextReader()
	if err != nil {
		return nil, err
	}
	tmpBuffReader := bufio.NewReader(r)
	cmdUrl, _, err := tmpBuffReader.ReadLine()
	if err == io.EOF {
		return nil, err
	}
	if err != nil {
		return nil, ErrReadCmd
	}

	msgId, _, err := tmpBuffReader.ReadLine()
	if err != nil {
		return nil, err
	}

	isFileStr, _, err := tmpBuffReader.ReadLine()
	if err != nil {
		return nil, ErrReadMsgType
	}

	isFile := true
	if string(isFileStr) == "0" {
		isFile = false
	}

	if isFile {
		filePath, _, err := tmpBuffReader.ReadLine()
		if err != nil {
			return nil, ErrReadMsgContent
		}
		websocketContext.msgFile, err = os.OpenFile(string(filePath), os.O_RDONLY, 0666)
		if err != nil {
			return nil, ErrReadMsgContent
		}
	} else {
		datLenStr, _, err := tmpBuffReader.ReadLine()
		if err != nil {
			return nil, ErrReadMsgContent
		}
		dataLen, err := strconv.ParseInt(string(datLenStr), 10, 64)
		if err != nil {
			return nil, ErrReadMsgContent
		}

		msgFile, err := this.readSizeContentToFile(tmpBuffReader, int(dataLen))
		if err != nil {
			return nil, err
		}
		websocketContext.DataLen = dataLen
		websocketContext.msgFile = msgFile
	}

	websocketContext.msgFileBuffReader = bufio.NewReader(websocketContext.msgFile)
	websocketContext.Cmd = string(cmdUrl)
	websocketContext.MsgId = string(msgId)
	websocketContext.Mod = string(isFileStr)
	websocketContext.wsConn = this.wsConn
	websocketContext.parseFields()
	if websocketContext.Err != nil {
		return nil, websocketContext.Err
	}
	return websocketContext, nil
}

func (this *engineHandle) readSizeContentToFile(currentBufferReader *bufio.Reader, size int) (*os.File, error) {
	tmpFile, err := ioutil.TempFile("devPlatform", "*")
	if err != nil {
		return nil, ErrCreateContentFile
	}

	totalReadSize := this.readLastBuf.Len()
	if totalReadSize >= size {
		lastData := this.readLastBuf.Bytes()
		writeData := lastData[:size]
		lastData = lastData[size:]
		_, err = tmpFile.Write(writeData)
		if err != nil {
			return nil, ErrReadMsgContent
		}

		this.readLastBuf.Reset()
		if len(lastData) > 0 {
			this.readLastBuf.Write(lastData)
		}
		return tmpFile, nil
	} else {
		_, err = tmpFile.Write(this.readLastBuf.Bytes())
		if err != nil {
			return nil, ErrReadMsgContent
		}
		this.readLastBuf.Reset()
	}

	for {
		readLen, err := currentBufferReader.Read(this.readBufSlice)
		if err == io.EOF {
			_, r, err := this.wsConn.NextReader()
			if err != nil {
				return nil, err
			}
			currentBufferReader = bufio.NewReader(r)
			continue
		}
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
			return nil, ErrReadMsgContent
		}

		if isOk {
			if otherSize > 0 {
				this.readLastBuf.Write(this.readBufSlice[readLen : readLen+otherSize])
			}
			return tmpFile, nil
		}

	}
}
