package conn

import (
	"bytes"
	"context"
	"errors"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/data"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const maxMsgSize = 1024 * 1024

type ModType string

const (
	ModTypeFile ModType = "0"
	ModTypeMem  ModType = "1"
)

type MsgInfo struct {
	Cmd        string
	Mod        ModType
	MsgId      string
	Data       string
	needReturn bool
	callback   chan *MsgInfo
	err        chan error
	context    *Context
}

func (this *MsgInfo) NeedReturn() bool {
	return this.needReturn
}

type ConnectionBuf struct {
	*websocket.Conn
	sendInfo      chan *MsgInfo
	connBufReader *bufWebsocketReader
	readLastBuf   *strings.Builder
	writeBuf      *bytes.Buffer
	readBufSlice  []byte
	writeBufSlice []byte
	msgMap        map[string]*MsgInfo
	lock          *sync.Mutex
}

func (this *ConnectionBuf) SendMsgAndReturnWithTimeOut(info *MsgInfo, timeout time.Duration) (*Context, error) {
	if this.sendInfo == nil {
		return nil, errors.New("连接已断开")
	}
	info.err = make(chan error, 1)
	info.callback = make(chan *MsgInfo, 1)
	defer func() {
		close(info.err)
		close(info.callback)
	}()

	this.sendInfo <- info
	err := <-info.err
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			delete(this.msgMap, info.MsgId)
			return nil, errors.New("超时")
		case callback := <-info.callback:
			if callback == nil {
				return nil, <-info.err
			}
			return NewWebSocketContext(this, callback.Cmd, info.NeedReturn(), callback.MsgId, callback.Mod, callback.Data), nil
		}
	}
}

func (this *ConnectionBuf) SendMsg(info *MsgInfo) error {
	if this.sendInfo == nil {
		return errors.New("连接已断开")
	}
	defer func() { recover() }()
	info.err = make(chan error, 1)
	defer func() { close(info.err) }()
	this.sendInfo <- info
	return <-info.err
}

func (this *ConnectionBuf) SendMsgAndReturn(info *MsgInfo) (*Context, error) {
	if this.sendInfo == nil {
		return nil, errors.New("连接已断开")
	}
	info.err = make(chan error, 1)
	info.callback = make(chan *MsgInfo, 1)
	defer func() { recover() }()
	defer func() {
		close(info.err)
		close(info.callback)
	}()

	this.sendInfo <- info
	err := <-info.err
	if err != nil {
		return nil, err
	}
	callback := <-info.callback
	if callback == nil {
		return nil, <-info.err

	}
	return NewWebSocketContext(this, callback.Cmd, info.NeedReturn(), callback.MsgId, callback.Mod, callback.Data), nil
}

func (this *ConnectionBuf) writeLoop() {
	for {
		info, isOpen := <-this.sendInfo
		if !isOpen {
			return
		}

		if info.MsgId == "" {
			uid, err := uuid.NewV4()
			if err != nil {
				info.err <- err
				continue
			}
			info.MsgId = uid.String() + strconv.FormatInt(time.Now().UnixNano(), 10)
		}

		needStr := "0"
		if info.callback != nil {
			this.msgMap[info.MsgId] = info
			needStr = "1"
		}

		this.writeBuf.Reset()
		this.writeBuf.WriteString(info.Cmd)
		this.writeBuf.WriteRune('\n')
		this.writeBuf.WriteString(needStr)
		this.writeBuf.WriteRune('\n')
		this.writeBuf.WriteString(info.MsgId)
		this.writeBuf.WriteRune('\n')
		this.writeBuf.WriteString(string(info.Mod))
		this.writeBuf.WriteRune('\n')
		err := this.WriteMessage(websocket.TextMessage, this.writeBuf.Bytes())
		if err != nil {
			info.err <- err
			continue
		}
		if info.Mod == "0" {
			err = this.WriteMessage(websocket.TextMessage, []byte(info.Data))
			if err != nil {
				info.err <- err
			}
			err = this.WriteMessage(websocket.TextMessage, []byte{'\n'})
			if err != nil {
				info.err <- err
			}
		} else {
			this.writeSendData(info)
		}
		info.err <- nil
	}
}

func (this *ConnectionBuf) writeSendData(info *MsgInfo) {
	file, err := os.OpenFile(info.Data, os.O_RDONLY, 0666)
	if err != nil {
		info.err <- err
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		info.err <- err
		return
	}

	if stat.Size() <= maxMsgSize {
		readLen, err := file.Read(this.writeBufSlice)
		if err != nil {
			info.err <- err
			return
		}
		if err = this.WriteMessage(websocket.TextMessage, bytes.Join([][]byte{
			[]byte(strconv.FormatInt(int64(readLen), 10)),
			{'\n'},
		}, nil)); err != nil {
			info.err <- err
			return
		}
		if err = this.WriteMessage(websocket.TextMessage, this.writeBufSlice[:readLen]); err != nil {
			info.err <- err
			return
		}
		return
	}

	blockSize := int(math.Ceil(float64(stat.Size()) / maxMsgSize))
	for i := 0; i < blockSize; i++ {
		readLen, err := file.Read(this.writeBufSlice)
		if err != nil {
			info.err <- err
			return
		}

		writeMsg := this.writeBufSlice[:readLen]
		if i == blockSize-1 {
			if writeMsg[readLen-1] != '\n' {
				writeMsg = bytes.Join([][]byte{
					writeMsg,
					{'\n'},
				}, nil)
			}
		}
		if err = this.WriteMessage(websocket.TextMessage, writeMsg); err != nil {
			if info.callback != nil {
				info.err <- err
			}
			return
		}
	}

}

func (this *ConnectionBuf) ReadMsgInfo() (*MsgInfo, error) {
ReadBegin:
	cmdUrl, err := this.connBufReader.ReadLine()
	if err != nil {
		return nil, err
	}

	needReturn, err := this.connBufReader.ReadLine()
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
	dataLen := int64(0)
	mod := ModTypeMem
	if isFile {
		filePath, err := this.connBufReader.ReadLine()
		if err != nil {
			return nil, err
		}
		msgFile = filePath
		mod = ModTypeFile
	} else {
		datLenStr, err := this.connBufReader.ReadLine()
		if err != nil {
			return nil, err
		}
		dataLen, err = strconv.ParseInt(datLenStr, 10, 64)
		if err != nil {
			return nil, err
		}

		msgFile, err = this.readSizeContentToFile(int(dataLen))
		if err != nil {
			return nil, err
		}
	}

	if cmdUrl == "" {
		msgInfo, ok := this.msgMap[msgId]
		if !ok {
			d, _ := data.MarshalErr("404", "未知的请求")
			this.SendMsg(&MsgInfo{
				MsgId: msgId,
				Mod:   mod,
				Data:  d,
			})
			goto ReadBegin
		}
		if msgInfo.callback != nil {
			delete(this.msgMap, msgId)
			msgInfo.Mod = mod
			msgInfo.MsgId = msgId
			msgInfo.Data = msgFile
			msgInfo.needReturn = needReturn == "1"
			msgInfo.callback <- msgInfo
		} else {
			d, _ := data.MarshalErr("404", "未知的返回异常")
			this.SendMsg(&MsgInfo{
				MsgId: msgId,
				Mod:   mod,
				Data:  d,
			})
		}
		goto ReadBegin
	}

	return &MsgInfo{
		Cmd:        cmdUrl,
		Mod:        mod,
		Data:       msgFile,
		MsgId:      msgId,
		needReturn: needReturn == "1",
	}, nil
}

func (this *ConnectionBuf) readSizeContentToFile(size int) (string, error) {
	tmpDir := filepath.Join(os.TempDir(), "devPlatform")
	os.MkdirAll(tmpDir, 0777)
	tmpFile, err := ioutil.TempFile(tmpDir, "*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	totalReadSize := this.readLastBuf.Len()
	if totalReadSize >= size {
		lastData := this.readLastBuf.String()
		writeData := lastData[:size]
		lastData = lastData[size:]
		_, err = tmpFile.WriteString(writeData)
		if err != nil {
			return "", err
		}

		this.readLastBuf.Reset()
		if len(lastData) > 0 {
			this.readLastBuf.WriteString(lastData)
		}
		return tmpFile.Name(), nil
	} else {
		_, err = tmpFile.WriteString(this.readLastBuf.String())
		if err != nil {
			return "", err
		}
		this.readLastBuf.Reset()
	}

	for {
		line, err := this.connBufReader.ReadLine()
		if err != nil {
			return "", err
		}
		line += "\n"
		lineLen := utf8.RuneCountInString(line)

		totalReadSize += lineLen
		isOk := false
		otherSize := totalReadSize - size
		if totalReadSize >= size {
			lineLen -= otherSize
			isOk = true
		}

		//readData := this.readBufSlice[:readLen]
		_, err = tmpFile.WriteString(line[:lineLen])
		if err != nil {
			return "", err
		}

		if isOk {
			if otherSize > 0 {
				this.readLastBuf.WriteString(line[lineLen : lineLen+otherSize])
			}
			return tmpFile.Name(), nil
		}

	}
}

func (this *ConnectionBuf) Close() {
	defer func() {
		recover()
	}()

	if len(this.msgMap) > 0 {
		for _, v := range this.msgMap {
			this.closeWaitMsg(v)
		}
	}

	this.Conn.Close()
	close(this.sendInfo)
	this.sendInfo = nil
}

func (this *ConnectionBuf) closeWaitMsg(msgInfo *MsgInfo) {
	defer func() { recover() }()
	if msgInfo.callback != nil {
		close(msgInfo.callback)
	}

	if msgInfo.err != nil {
		msgInfo.err <- errors.New("连接被关闭")
		close(msgInfo.err)
	}

}

func NewConnectionBuf(wsConn *websocket.Conn) *ConnectionBuf {
	connBuf := &ConnectionBuf{
		Conn:          wsConn,
		sendInfo:      make(chan *MsgInfo, 1),
		connBufReader: newBufWebsocketReader(wsConn),
		readLastBuf:   &strings.Builder{},
		writeBuf:      &bytes.Buffer{},
		readBufSlice:  make([]byte, maxMsgSize, maxMsgSize),
		writeBufSlice: make([]byte, maxMsgSize, maxMsgSize),
		msgMap:        make(map[string]*MsgInfo),
	}

	go connBuf.writeLoop()
	return connBuf
}
