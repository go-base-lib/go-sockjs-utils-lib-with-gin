package conn

import (
	"bytes"
	"context"
	"errors"
	"github.com/go-base-lib/go-sockjs-utils-lib-with-gin/sockjs/data"
	"github.com/go-base-lib/logs"
	"github.com/gofrs/uuid"
	"github.com/igm/sockjs-go/v3/sockjs"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxMsgSize = 1024 * 1024

var memDataEndFlag = []byte{'\r', '\n', '0', '\r', '\n'}

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
	if this == nil {
		return false
	}
	return this.needReturn
}

type ConnectionBuf struct {
	*sockjs.Session
	connFlag      string
	sendInfo      chan *MsgInfo
	connBufReader *bufWebsocketReader
	readLastBuf   *strings.Builder
	writeBuf      *bytes.Buffer
	readBufSlice  []byte
	writeBufSlice []byte
	msgMap        map[string]*MsgInfo
	lock          *sync.Mutex
}

func (this *ConnectionBuf) GetConnFlag() string {
	if this == nil {
		return ""
	}
	return this.connFlag
}

func (this *ConnectionBuf) SettingConnFlag(flag string) {
	if this == nil {
		return
	}
	this.connFlag = flag
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
	if this == nil || this.sendInfo == nil {
		return errors.New("连接已断开")
	}
	defer func() { recover() }()
	info.err = make(chan error, 1)
	defer func() { close(info.err) }()
	this.sendInfo <- info
	return <-info.err
}

func (this *ConnectionBuf) SendMsgAndReturn(info *MsgInfo) (*Context, error) {
	if this == nil || this.sendInfo == nil {
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
		err := this.Session.Send(this.writeBuf.String())
		if err != nil {
			info.err <- err
			continue
		}
		if info.Mod == "0" {
			err = this.Session.Send(info.Data)
			if err != nil {
				info.err <- err
			}
			err = this.Session.Send("\n")
			if err != nil {
				info.err <- err
			}
			if logs.CurrentLevel() >= logrus.DebugLevel {
				f, e := ioutil.ReadFile(info.Data)
				if e == nil {
					logs.Debugf("命令[%s], 模式[%s], 数据将被发送: \n%s", info.Cmd, info.Mod, string(f))
				}
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
	defer os.RemoveAll(info.Data)
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		info.err <- err
		return
	}

	if logs.CurrentLevel() >= logrus.DebugLevel {
		f, e := ioutil.ReadFile(info.Data)
		if e == nil {
			logs.Debugf("命令[%s], 模式[%s], 数据将被发送: \n%s", info.Cmd, info.Mod, string(f))
		}
	}

	if stat.Size() <= maxMsgSize {
		readLen, err := file.Read(this.writeBufSlice)
		if err != nil {
			info.err <- err
			return
		}

		sendData := this.writeBufSlice[:readLen]
		//sendDataLen := utf8.RuneCountInString(string(sendData))
		//if err = this.Session.Send(strings.Join([]string{
		//	strconv.FormatInt(int64(sendDataLen), 10),
		//	"\n",
		//}, "")); err != nil {
		//	info.err <- err
		//	return
		//}
		if err = this.Session.Send(string(sendData)); err != nil {
			info.err <- err
			return
		}
	} else {
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

			if err = this.Session.Send(string(writeMsg)); err != nil {
				if info.callback != nil {
					info.err <- err
				}
				return
			}
		}
	}

	if err = this.Session.Send(string(memDataEndFlag)); err != nil {
		if info.callback != nil {
			info.err <- err
		}
		return
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
	mod := ModTypeMem
	if isFile {
		filePath, err := this.connBufReader.ReadLine()
		if err != nil {
			return nil, err
		}
		msgFile = filePath
		mod = ModTypeFile
	} else {
		//datLenStr, err := this.connBufReader.ReadLine()
		//if err != nil {
		//	return nil, err
		//}
		//dataLen, err = strconv.ParseInt(datLenStr, 10, 64)
		//if err != nil {
		//	return nil, err
		//}
		msgFile, err = this.readSizeContentToFile()
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

func (this *ConnectionBuf) readSizeContentToFile() (string, error) {
	tmpDir := filepath.Join(os.TempDir(), "devPlatform")
	os.MkdirAll(tmpDir, 0777)
	tmpFile, err := ioutil.TempFile(tmpDir, "*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	for {
		readLen, err := this.connBufReader.Read(this.readBufSlice)
		if err != nil {
			return "", err
		}
		readData := this.readBufSlice[:readLen]
		if bytes.Equal(readData, memDataEndFlag) {
			return tmpFile.Name(), nil
		}

		if _, err = tmpFile.Write(readData); err != nil {
			return "", err
		}
	}

	//totalReadSize := this.readLastBuf.Len()
	//if totalReadSize >= size {
	//	lastData := this.readLastBuf.String()
	//	writeData := lastData[:size]
	//	lastData = lastData[size:]
	//	_, err = tmpFile.WriteString(writeData)
	//	if err != nil {
	//		return "", err
	//	}
	//
	//	this.readLastBuf.Reset()
	//	if len(lastData) > 0 {
	//		this.readLastBuf.WriteString(lastData)
	//	}
	//	return tmpFile.Name(), nil
	//} else {
	//	_, err = tmpFile.WriteString(this.readLastBuf.String())
	//	if err != nil {
	//		return "", err
	//	}
	//	this.readLastBuf.Reset()
	//}
	//
	//for {
	//	line, err := this.connBufReader.ReadLine()
	//	if err != nil {
	//		return "", err
	//	}
	//	line += "\n"
	//	lineRune := []rune(line)
	//	lineLen := len(lineRune)
	//
	//	totalReadSize += lineLen
	//	isOk := false
	//	otherSize := totalReadSize - size
	//	if totalReadSize >= size {
	//		lineLen -= otherSize
	//		isOk = true
	//	}
	//
	//	//readData := this.readBufSlice[:readLen]
	//	_, err = tmpFile.WriteString(string(lineRune[:lineLen]))
	//	if err != nil {
	//		return "", err
	//	}
	//
	//	if isOk {
	//		if otherSize > 0 {
	//			runeLine := lineRune[lineLen:]
	//			if runeLine[0] == '\n' {
	//				runeLine = runeLine[1:]
	//			}
	//			this.readLastBuf.WriteString(string(runeLine))
	//		}
	//		return tmpFile.Name(), nil
	//	}

	//}
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

	this.Session.Close(200, "success")
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

func NewConnectionBuf(wsConn *sockjs.Session) *ConnectionBuf {
	connBuf := &ConnectionBuf{
		Session:       wsConn,
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
