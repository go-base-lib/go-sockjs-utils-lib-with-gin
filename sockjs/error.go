package sockjs

import "errors"

var (
	ErrReadCmd           = errors.New("读取Cmd失败")
	ErrNoCmd             = errors.New("没有找到对应指令")
	ErrReadMsgType       = errors.New("读取消息片段类型失败")
	ErrReadMsgContent    = errors.New("读取消息内容失败")
	ErrCreateContentFile = errors.New("创建消息文件失败")
)
