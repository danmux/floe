package log

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
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

func prefix(level string, args ...interface{}) []interface{} {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	} else {
		bits := strings.Split(file, "/")
		if len(bits) > 2 {
			file = bits[len(bits)-2] + "/" + bits[len(bits)-1]
		}
	}
	a := []interface{}{level, fmt.Sprintf("(%s:%d)", file, line)}
	a = append(a, args...)
	return a
}

func Debug(args ...interface{}) {
	if badLevel(lDbg) {
		return
	}
	args = prefix(dbg, args...)
	logger.Println(args...)
}

func Debugf(format string, args ...interface{}) {
	if badLevel(lDbg) {
		return
	}
	args = []interface{}{fmt.Sprintf(format, args...)}
	args = prefix(dbg, args...)
	logger.Println(args...)
}

func Info(args ...interface{}) {
	if badLevel(lInf) {
		return
	}
	args = prefix(inf, args...)
	logger.Println(args...)
}

func Infof(format string, args ...interface{}) {
	if badLevel(lInf) {
		return
	}
	args = []interface{}{fmt.Sprintf(format, args...)}
	args = prefix(inf, args...)
	logger.Println(args...)
}

func Warning(args ...interface{}) {
	if badLevel(lWar) {
		return
	}
	args = prefix(war, args...)
	logger.Println(args...)
}

func Error(args ...interface{}) {
	if badLevel(lErr) {
		return
	}
	args = prefix(err, args...)
	logger.Println(args...)
}

func Errorf(format string, args ...interface{}) {
	if badLevel(lErr) {
		return
	}
	args = []interface{}{fmt.Sprintf(format, args...)}
	args = prefix(err, args...)
	logger.Println(args...)
}

func Fatal(args ...interface{}) {
	if badLevel(lErr) {
		return
	}
	args = prefix(err, args...)
	logger.Println(args...)
	os.Exit(255)
}
