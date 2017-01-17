package log

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const (
	dbg  = "[D]"
	inf  = "[I]"
	err  = "[E]"
	war  = "[W]"
	pref = ""
	form = log.Ldate | log.Lmicroseconds

	lErr = 3
	lWar = 4
	lInf = 6
	lDbg = 7
)

var (
	logger *log.Logger
	logbuf bytes.Buffer
	level  int = 7
	mu     sync.Mutex
)

// 3 = error
// 4 = warning
// 7 = debug

func init() {
	NewStdErrLogger()
}

func badLevel(l int) bool {
	mu.Lock()
	b := level < l
	mu.Unlock()
	return b
}

func SetLevel(l int) {
	mu.Lock()
	level = l
	mu.Unlock()
}

func NewStdErrLogger() {
	logger = log.New(os.Stderr, pref, form)
}

func NewCaptureLogger() {
	logger = log.New(&logbuf, pref, form)
}

func NewSilentLogger() {
	logger = log.New(ioutil.Discard, pref, form)
}

func PrintLog() {
	fmt.Print(&logbuf)
}

func Debug(args ...interface{}) {
	if badLevel(lDbg) {
		return
	}
	args = append([]interface{}{dbg}, args...)
	logger.Println(args...)
}

func Debugf(format string, args ...interface{}) {
	if badLevel(lDbg) {
		return
	}
	logger.Println(dbg, fmt.Sprintf(format, args...))
}

func Info(args ...interface{}) {
	if badLevel(lInf) {
		return
	}
	args = append([]interface{}{inf}, args...)
	logger.Println(args...)
}

func Infof(format string, args ...interface{}) {
	if badLevel(lInf) {
		return
	}
	logger.Println(inf, fmt.Sprintf(format, args...))
}

func Warning(args ...interface{}) {
	if badLevel(lWar) {
		return
	}
	args = append([]interface{}{war}, args...)
	logger.Println(args...)
}

func Error(args ...interface{}) {
	if badLevel(lErr) {
		return
	}
	args = append([]interface{}{err}, args...)
	logger.Println(args...)
}

func Fatal(args ...interface{}) {
	if badLevel(lErr) {
		return
	}
	args = append([]interface{}{err}, args...)
	logger.Println(args...)
	os.Exit(255)
}

func Errorf(format string, args ...interface{}) {
	if badLevel(lErr) {
		return
	}
	logger.Println(err, fmt.Sprintf(format, args...))
}

// func V(l int) bool {
// 	return l < level
// }

// func Info(args ...interface{}) {
// 	glog.InfoDepth(1, args...)
// }

// func Infof(format string, args ...interface{}) {
// 	glog.InfoDepth(1, fmt.Sprintf(format, args...))
// }

// func Error(args ...interface{}) {
// 	glog.ErrorDepth(1, args...)
// }

// func Warning(args ...interface{}) {
// 	glog.WarningDepth(1, args...)
// }

// func Fatal(args ...interface{}) {
// 	glog.FatalDepth(1, args...)
// }

// func Errorf(format string, args ...interface{}) {
// 	glog.ErrorDepth(1, fmt.Sprintf(format, args...))
// }

// func V(level glog.Level) glog.Verbose {
// 	return glog.V(level)
// }
