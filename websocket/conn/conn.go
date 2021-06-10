package conn

import (
	"bytes"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/data"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"time"
)

const maxMsgSize = 1024 * 1024

type MsgInfo struct {
	Cmd      string
	Mod      string
	MsgId    string
	Len      int64
	Data     string
	callback chan *MsgInfo
	err      chan error
}

type ConnectionBuf struct {
	*websocket.Conn
	sendInfo      chan *MsgInfo
	connBufReader *bufWebsocketReader
	readLastBuf   *bytes.Buffer
	writeBuf      *bytes.Buffer
	readBufSlice  []byte
	writeBufSlice []byte
	msgMap        map[string]*MsgInfo
}

func (this *ConnectionBuf) SendMsg(info *MsgInfo) error {
	info.err = make(chan error, 1)
	defer func() { close(info.err) }()
	this.sendInfo <- info
	return <-info.err
}

func (this *ConnectionBuf) SendMsgAndReturn(info *MsgInfo) (*Context, error) {
	info.err = make(chan error, 1)
	info.callback = make(chan *MsgInfo, 1)
	defer func() {
		close(info.err)
		close(info.callback)
	}()

	err := <-info.err
	if err != nil {
		return nil, err
	}
	callback := <-info.callback
	if callback == nil {
		return nil, <-info.err

	}
	return NewWebSocketContext(this, callback.Cmd, callback.MsgId, callback.Mod, callback.Data), nil
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

		if info.callback != nil {
			this.msgMap[info.MsgId] = info
		}

		this.writeBuf.Reset()
		this.writeBuf.WriteString(info.Cmd)
		this.writeBuf.WriteRune('\n')
		this.writeBuf.WriteString(info.MsgId)
		this.writeBuf.WriteRune('\n')
		this.writeBuf.WriteString(info.Mod)
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
	if isFile {
		filePath, err := this.connBufReader.ReadLine()
		if err != nil {
			return nil, err
		}
		msgFile = filePath
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
				Mod:   isFileStr,
				Data:  d,
			})
			goto ReadBegin
		}
		if msgInfo.callback != nil {
			delete(this.msgMap, msgId)
			msgInfo.Mod = isFileStr
			msgInfo.Len = dataLen
			msgInfo.MsgId = msgId
			msgInfo.Data = msgFile
			msgInfo.callback <- msgInfo
		} else {
			d, _ := data.MarshalErr("404", "未知的返回异常")
			this.SendMsg(&MsgInfo{
				MsgId: msgId,
				Mod:   isFileStr,
				Data:  d,
			})
		}
		goto ReadBegin
	}

	return &MsgInfo{
		Cmd:   cmdUrl,
		Mod:   isFileStr,
		Data:  msgFile,
		Len:   dataLen,
		MsgId: msgId,
	}, nil
}

func (this *ConnectionBuf) readSizeContentToFile(size int) (string, error) {
	tmpFile, err := ioutil.TempFile("devPlatform", "*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	totalReadSize := this.readLastBuf.Len()
	if totalReadSize >= size {
		lastData := this.readLastBuf.Bytes()
		writeData := lastData[:size]
		lastData = lastData[size:]
		_, err = tmpFile.Write(writeData)
		if err != nil {
			return "", err
		}

		this.readLastBuf.Reset()
		if len(lastData) > 0 {
			this.readLastBuf.Write(lastData)
		}
		return tmpFile.Name(), nil
	} else {
		_, err = tmpFile.Write(this.readLastBuf.Bytes())
		if err != nil {
			return "", err
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
			return "", err
		}

		if isOk {
			if otherSize > 0 {
				this.readLastBuf.Write(this.readBufSlice[readLen : readLen+otherSize])
			}
			return tmpFile.Name(), nil
		}

	}
}

func (this *ConnectionBuf) Close() {
	defer func() {
		recover()
	}()
	this.Conn.Close()
	close(this.sendInfo)
}

func NewConnectionBuf(wsConn *websocket.Conn) *ConnectionBuf {
	connBuf := &ConnectionBuf{
		Conn:          wsConn,
		sendInfo:      make(chan *MsgInfo, 1),
		connBufReader: newBufWebsocketReader(wsConn),
		readLastBuf:   &bytes.Buffer{},
		writeBuf:      &bytes.Buffer{},
		readBufSlice:  make([]byte, maxMsgSize, maxMsgSize),
		writeBufSlice: make([]byte, maxMsgSize, maxMsgSize),
		msgMap:        make(map[string]*MsgInfo),
	}

	go connBuf.writeLoop()
	return connBuf
}
