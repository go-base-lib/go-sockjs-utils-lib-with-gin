package logs


type SocketLogs interface {
	Debug(args ...interface{})
	Debugln(args ...interface{})
	DebugF(format string, args ...interface{})
	Info(args ...interface{})
	Infoln(args ...interface{})
	InfoF(format string, args ...interface{})
	Warn(args ...interface{})
	Warnln(args ...interface{})
	WarnF(format string, args ...interface{})
	Error(args ...interface{})
	Errorln(args ...interface{})
	ErrorF(format string, args ...interface{})
}
