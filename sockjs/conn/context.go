package conn

import (
	"bufio"
	"errors"
	"github.com/devloperPlatform/go-sockjs-utils-lib-with-gin/sockjs/data"
	"github.com/devloperPlatform/go-sockjs-utils-lib-with-gin/sockjs/logs"
	"os"
	"time"
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
	mod ModType
	// 消息文件路径
	msgFilePath string
	// 错误
	Err error
	// 返回需要接收的消息超时时间
	ReturnRecvMsgTimeout time.Duration
	// 字段位置和长度
	fieldInfoMap map[string]*data.FieldInfo
	// 连接
	wsConn *ConnectionBuf
	// 是否没有返回值
	isVoid bool
	// 是否已经返回
	isReturn bool
	// 是否需要返回
	needReturn bool
	children   []*Context
}

func (this *Context) IsReturn() bool {
	return this.isReturn
}

func (this *Context) NeedReturn() bool {
	return this.needReturn
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

func (this *Context) ReturnData(v interface{}) error {
	f, err := data.Marshal(v)
	if err != nil {
		return err
	}

	stat, err := os.Stat(f)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return errors.New("传输文件失败")
	}
	return this.returnMsg(f)
}

func (this *Context) ReturnVoid() error {
	str, err := data.MarshalVoidStr()
	if err != nil {
		return err
	}
	return this.returnMsg(str)
}

func (this *Context) ReturnErr(code, msg string) error {
	f, err := data.MarshalErr(code, msg)
	if err != nil {
		return err
	}
	return this.returnMsg(f)
}

func (this *Context) ReturnVoidAndRecv() (*Context, error) {
	str, err := data.MarshalVoidStr()
	if err != nil {
		return nil, err
	}
	return this.returnMsgAndRecv(str)
}

func (this *Context) ReturnDataAndRecv(v interface{}) (*Context, error) {
	f, err := data.Marshal(v)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(f)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, errors.New("传输文件失败")
	}
	return this.returnMsgAndRecv(f)
}

func (this *Context) ReturnErrAndRecv(code, msg string) (*Context, error) {
	f, err := data.MarshalErr(code, msg)
	if err != nil {
		return nil, err
	}
	return this.returnMsgAndRecv(f)
}

func (this *Context) returnMsgAndRecv(f string) (*Context, error) {
	var (
		c   *Context
		err error

		msg = &MsgInfo{
			Mod:   this.mod,
			MsgId: this.MsgId(),
			Data:  f,
		}
	)

	this.isReturn = true
	if this.ReturnRecvMsgTimeout > 0 {
		c, err = this.wsConn.SendMsgAndReturnWithTimeOut(msg, this.ReturnRecvMsgTimeout)
	} else {
		c, err = this.wsConn.SendMsgAndReturn(msg)
	}
	if err != nil {
		return nil, err
	}
	if this.children == nil {
		this.children = make([]*Context, 0, 8)
		this.children = append(this.children, c)
	}

	if c.HaveErr() {
		return c, c.Err
	}
	return c, err

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

func (this *Context) returnMsg(f string) error {
	if this.mod == "" {
		return errors.New("空Context只能发送数据无法返回数据")
	}
	if !this.needReturn {
		return errors.New("消息无需返回")
	}
	if this.isReturn {
		return errors.New("消息已经返回请勿多次返回")
	}
	this.isReturn = true
	return this.wsConn.SendMsg(&MsgInfo{
		Mod:   this.mod,
		MsgId: this.MsgId(),
		Data:  f,
	})
}

func (this *Context) Cmd() string {
	return this.cmd
}

func (this *Context) MsgFilePath() string {
	return this.msgFilePath
}

func (this *Context) IsVoid() bool {
	return this.isVoid
}

func (this *Context) Mod() ModType {
	return this.mod
}

func (this *Context) MsgId() string {
	return this.msgId
}

func (this *Context) HaveErr() bool {
	return this.Err != nil
}

func (this *Context) SendVoidMsg(cmd string) error {
	f, err := data.MarshalVoidStr()
	if err != nil {
		return err
	}
	return this.wsConn.SendMsg(&MsgInfo{
		Cmd:  cmd,
		Mod:  ModTypeMem,
		Data: f,
	})
}

func (this *Context) SendVoidMsgAndReturn(cmd string) (*Context, error) {
	f, err := data.MarshalVoidStr()
	if err != nil {
		return nil, err
	}

	c, err := this.wsConn.SendMsgAndReturn(&MsgInfo{
		Cmd:  cmd,
		Mod:  ModTypeMem,
		Data: f,
	})
	if err != nil {
		return nil, err
	}
	if this.children == nil {
		this.children = make([]*Context, 0, 8)
		this.children = append(this.children, c)
	}

	if c.HaveErr() {
		return c, c.Err
	}

	return c, nil
}

func (this *Context) SendVoidMsgAndReturnWithTimeout(cmd string, timeout time.Duration) (*Context, error) {
	f, err := data.MarshalVoidStr()
	if err != nil {
		return nil, err
	}

	c, err := this.wsConn.SendMsgAndReturnWithTimeOut(&MsgInfo{
		Cmd:  cmd,
		Mod:  ModTypeMem,
		Data: f,
	}, timeout)
	if err != nil {
		return nil, err
	}
	if this.children == nil {
		this.children = make([]*Context, 0, 8)
		this.children = append(this.children, c)
	}

	if c.HaveErr() {
		return c, c.Err
	}

	return c, nil
}

func (this *Context) SendMsg(cmd string, modType ModType, sendData interface{}) error {
	var (
		f   string
		err error
	)

	if sendData == nil {
		f, err = data.MarshalVoidStr()
	} else {
		f, err = data.Marshal(sendData)
	}
	if err != nil {
		return err
	}

	return this.wsConn.SendMsg(&MsgInfo{
		Cmd:  cmd,
		Mod:  modType,
		Data: f,
	})
}

func (this *Context) SendMsgAndReturnWithTimeout(cmd string, modType ModType, sendData interface{}, timeout time.Duration) (*Context, error) {
	var (
		f   string
		err error
	)

	if sendData == nil {
		f, err = data.MarshalVoidStr()
	} else {
		f, err = data.Marshal(sendData)
	}
	if err != nil {
		return nil, err
	}
	c, err := this.wsConn.SendMsgAndReturnWithTimeOut(&MsgInfo{
		Cmd:  cmd,
		Mod:  modType,
		Data: f,
	}, timeout)
	if err != nil {
		return nil, err
	}
	if this.children == nil {
		this.children = make([]*Context, 0, 8)
		this.children = append(this.children, c)
	}

	if c.HaveErr() {
		return c, c.Err
	}
	return c, err
}

func (this *Context) SendMsgAndReturn(cmd string, modType ModType, sendData interface{}) (*Context, error) {
	var (
		f   string
		err error
	)

	if sendData == nil {
		f, err = data.MarshalVoidStr()
	} else {
		f, err = data.Marshal(sendData)
	}
	if err != nil {
		return nil, err
	}
	c, err := this.wsConn.SendMsgAndReturn(&MsgInfo{
		Cmd:  cmd,
		Mod:  modType,
		Data: f,
	})
	if err != nil {
		return nil, err
	}
	if this.children == nil {
		this.children = make([]*Context, 0, 8)
		this.children = append(this.children, c)
	}

	if c.HaveErr() {
		return c, c.Err
	}
	return c, err
}

func (this *Context) GetConnFlag() string {
	return this.wsConn.GetConnFlag()
}

func (this *Context) SettingConnFlag(flag string) {
	this.wsConn.SettingConnFlag(flag)
}

func (this *Context) Destroy() {
	logs.LogRecord(logs.Debug, func(log logs.SocketLogs) {
		log.DebugF("命令[%s], 消息ID[%S] 正在被销毁", this.cmd, this.mod)
	})
	os.RemoveAll(this.msgFilePath)
	this.cmd = ""
	this.mod = ""
	this.msgFilePath = ""
	this.wsConn = nil
	this.fieldInfoMap = nil
	if this.needReturn && !this.IsReturn() {
		this.ReturnVoid()
	}
	for _, c := range this.children {
		c.Destroy()
	}
}

func (this *Context) CloseConn() {
	this.wsConn.Close()
}

func NewWebSocketContext(wsConn *ConnectionBuf, cmd string, needReturn bool, msgId string, mod ModType, filePath string) *Context {
	context := &Context{
		wsConn:      wsConn,
		cmd:         cmd,
		msgId:       msgId,
		mod:         mod,
		needReturn:  needReturn,
		msgFilePath: filePath,
	}
	if len(filePath) > 0 {
		context.parseFields()
	}
	return context
}
