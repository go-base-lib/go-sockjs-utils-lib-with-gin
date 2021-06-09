package socket

import (
	"bufio"
	"errors"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/conn"
	"github.com/devloperPlatform/go-websocket-utils-lib-with-gin/websocket/data"
	"os"
)

type readMsgFn func(msgReader *FieldMsgReader)

type FieldMsgReader struct {
	*data.FieldInfo
	fp string
}

func (this *FieldMsgReader) openFile(fn func(f *os.File) error) error {
	f, err := os.OpenFile(this.fp, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

func (this *FieldMsgReader) ReadAll() ([]byte, error) {
	buf := make([]byte, this.Len(), this.Len())
	err := this.openFile(func(f *os.File) error {
		_, err := f.Read(buf)
		return err
	})
	return buf, err
}

func NewFieldMsgReader(fieldInfo *data.FieldInfo, fp string) *FieldMsgReader {
	return &FieldMsgReader{
		FieldInfo: fieldInfo,
		fp:        fp,
	}
}

type Context struct {
	// Cmd 命令
	cmd string
	// 消息ID
	msgId string
	// 传输模式
	mod string
	// 消息文件路径
	msgFilePath string
	// 错误
	Err error
	// 字段位置和长度
	fieldInfoMap map[string]*data.FieldInfo
	// 连接
	wsConn *conn.ConnectionBuf
	// 是否没有返回值
	isVoid bool
}

func (this *Context) parseFields() {
	f, err := os.OpenFile(this.msgFilePath, os.O_RDONLY, 0666)
	if err != nil {
		this.Err = err
		return
	}
	defer f.Close()

	bufReader := bufio.NewReader(f)
	headerByte, err := bufReader.ReadByte()
	if err != nil {
		this.Err = err
		return
	}

	headerStr := string(headerByte)
	if data.IsVoid(headerStr) {
		this.isVoid = true
		return
	}

	if data.IsErr(headerStr) {
		_, err = f.Seek(0, 0)
		if err != nil {
			this.Err = err
			return
		}
		errData, err := data.Unmarshal2Err(f)
		if err != nil {
			this.Err = err
			return
		}
		this.Err = errData
		return
	}

	this.fieldInfoMap, err = data.Unmarshal2FieldInfoMap(this.msgFilePath)
	if err != nil {
		this.Err = err
	}

}

func (this *Context) Unmarshal(v interface{}) error {
	return data.UnmarshalByFilePath(this.msgFilePath, v)
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
	//buffer := bytes.Buffer{}
	//buffer.WriteRune('\n')
	//buffer.WriteString(this.MsgId)
	//buffer.WriteRune('\n')
	//buffer.WriteString(this.Mod)
	//buffer.WriteRune('\n')
	//buffer.WriteString(f)
	//buffer.WriteRune('\n')
	//defer buffer.Reset()
	//if err := this.wsConn.WriteMessage(websocket.TextMessage, buffer.Bytes()); err != nil {
	//	return err
	//}
	//return nil
	return nil
}

func (this *Context) Cmd() string {
	return this.cmd
}

func NewWebSocketContext(wsConn *conn.ConnectionBuf, cmd string, msgId string, mod string, filePath string) *Context {
	context := &Context{
		wsConn:      wsConn,
		cmd:         cmd,
		msgId:       msgId,
		mod:         mod,
		msgFilePath: filePath,
	}
	context.parseFields()
	return context
}
