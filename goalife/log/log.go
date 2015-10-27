// Package log provides an interface to logging that makes it easy to switch
// the logging on and off by replacing the value of the variable providing it.
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

// Null is a Logger that simply returns without evaluating its arguments.
// This is more efficient than using log.Logger.SetOutput(ioutil.Discard())
// because the discard approach still evaluates arguments and constructs
// the log message that gets thrown away.
func Null() Logger {
	return nullLogger{}
}

// Real is a shortcut to log.New with some useful defaults.
func Real() Logger {
	return log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
}
