package log

import "log"
import "os"

type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type nullLogger struct{}

func (_ nullLogger) Printf(format string, v ...interface{}) { return }
func (_ nullLogger) Println(v ...interface{})               { return }

func Null() Logger {
	return nullLogger{}
}

func Real() Logger {
	return log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}
