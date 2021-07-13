package logs

var instance SocketLogs = &defaultSocketLogs{}

type defaultSocketLogs struct {
}

func (d *defaultSocketLogs) Debug(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) Debugln(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) DebugF(format string, args ...interface{}) {
	return
}

func (d *defaultSocketLogs) Info(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) Infoln(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) InfoF(format string, args ...interface{}) {
	return
}

func (d *defaultSocketLogs) Warn(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) Warnln(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) WarnF(format string, args ...interface{}) {
	return
}

func (d *defaultSocketLogs) Error(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) Errorln(args ...interface{}) {
	return
}

func (d *defaultSocketLogs) ErrorF(format string, args ...interface{}) {
	return
}

type LogLevel uint32

var level LogLevel = Debug

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

func SetSocketLog(log SocketLogs) {
	if log == nil {
		return
	}
	instance = log
}

func GetSocketLog() SocketLogs {
	return instance
}

func SetLevel(l LogLevel) {
	level = l
}

func GetLevel() LogLevel {
	return level
}

func LogRecord(level LogLevel, handler func(log SocketLogs)) {
	if GetLevel() <= level {
		handler(GetSocketLog())
	}
}
