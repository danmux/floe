package glog

import (
	"fmt"
	"github.com/golang/glog"
)

func Info(args ...interface{}) {
	fmt.Println(args)
	glog.Info(args)
}

func Error(args ...interface{}) {
	glog.Info(args)
}

func Warning(args ...interface{}) {
	glog.Warning(args)
}
