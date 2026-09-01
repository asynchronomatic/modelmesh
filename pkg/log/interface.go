package log

type Logger interface {
	Printf(s string, v ...interface{})
	Panicf(s string, v ...interface{})
	Fatalf(s string, v ...interface{})
	Infof(s string, v ...interface{})
	Eventf(s string, v ...interface{})
	Warnf(s string, v ...interface{})
	Debugf(s string, v ...interface{})
	Highlightf(s string, v ...interface{})
	Errorf(s string, v ...interface{})
}
