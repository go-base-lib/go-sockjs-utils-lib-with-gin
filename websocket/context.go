package websocket

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"os"
	"strconv"
)

type FieldPositionAndSizeInfo struct {
	StartPos int64
	EndPos   int64
	Size     int64
}

type Context struct {
	// Cmd 命令
	Cmd string
	// msgFile 消息文件
	msgFile *os.File
	// msgFileBuffReader 文件buffer
	msgFileBuffReader *bufio.Reader
	// 消息ID
	MsgId string
	// 传输模式
	Mod string
	// 数据长度
	DataLen int64
	// 错误
	Err error
	// 字段名称
	FieldNameList []string
	// 字段位置和长度
	fieldPositionAndLen map[string]*FieldPositionAndSizeInfo
	// 连接
	wsConn *websocket.Conn
}

func (this *Context) parseFields() {
	this.FieldNameList = make([]string, 0, 8)
	this.fieldPositionAndLen = make(map[string]*FieldPositionAndSizeInfo)
	_, err := this.msgFile.Seek(0, 0)
	if err != nil {
		this.Err = err
		return
	}
	for {
		fieldName, _, err := this.msgFileBuffReader.ReadLine()
		if err == io.EOF {
			return
		}
		if err != nil {
			this.Err = err
			return
		}

		fieldLenStr, _, err := this.msgFileBuffReader.ReadLine()
		if err != nil {
			this.Err = err
			return
		}

		fieldLen, err := strconv.ParseInt(string(fieldLenStr), 10, 64)
		if err != nil {
			this.Err = err
			return
		}
		startPos, err := this.msgFile.Seek(0, 1)
		if err != nil {
			this.Err = err
			return
		}

		endPos, err := this.msgFile.Seek(fieldLen, 1)
		if err != nil {
			this.Err = err
			return
		}
		fieldNameStr := string(fieldName)
		this.FieldNameList = append(this.FieldNameList, fieldNameStr)
		this.fieldPositionAndLen[fieldNameStr] = &FieldPositionAndSizeInfo{
			StartPos: startPos,
			EndPos:   endPos,
			Size:     fieldLen,
		}
	}

}

func (this *Context) ReturnData(f string) error {
	stat, err := os.Stat(f)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return errors.New("传输文件失败")
	}
	return this.returnFileMsg(f)
}

func (this *Context) returnFileMsg(f string) error {
	buffer := bytes.Buffer{}
	buffer.WriteRune('\n')
	buffer.WriteString(this.MsgId)
	buffer.WriteRune('\n')
	buffer.WriteString(this.Mod)
	buffer.WriteRune('\n')
	buffer.WriteString(f)
	buffer.WriteRune('\n')
	defer buffer.Reset()
	if err := this.wsConn.WriteMessage(websocket.TextMessage, buffer.Bytes()); err != nil {
		return err
	}
	return nil
}
