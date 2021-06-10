package conn

import (
	"bufio"
	"errors"
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

func (this *FieldMsgReader) Field(name string) *FieldMsgReader {
	if this.Children() == nil {
		return nil
	}

	info, ok := this.Children()[name]
	if !ok {
		return nil
	}
	return NewFieldMsgReader(info, this.fp)
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
	wsConn *ConnectionBuf
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

func (this *Context) ReturnData(v interface{}) (*Context, error) {
	f, err := data.Marshal(v)
	if err != nil {
		return nil, err
	}
	data.Marshal(v)
	stat, err := os.Stat(f)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, errors.New("传输文件失败")
	}
	return this.returnFileMsg(f)
}

func (this *Context) ReturnErr(code, msg string) (*Context, error) {
	f, err := data.MarshalErr(code, msg)
	if err != nil {
		return nil, err
	}
	return this.returnFileMsg(f)
}

func (this *Context) FieldLen() int {
	return len(this.fieldInfoMap)
}

func (this *Context) Field(name string) *FieldMsgReader {
	info, ok := this.fieldInfoMap[name]
	if !ok {
		return nil
	}
	return NewFieldMsgReader(info, this.msgFilePath)
}

func (this *Context) returnFileMsg(f string) (*Context, error) {
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
	return nil, nil
}

func (this *Context) Cmd() string {
	return this.cmd
}

func (this *Context) IsVoid() bool {
	return this.isVoid
}

func (this *Context) Mod() string {
	return this.mod
}

func (this *Context) MsgId() string {
	return this.msgId
}

func (this *Context) HaveErr() bool {
	return this.Err != nil
}

func NewWebSocketContext(wsConn *ConnectionBuf, cmd string, msgId string, mod string, filePath string) *Context {
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
