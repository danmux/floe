package glog

import "github.com/golang/glog"

func Info(args ...interface{}) {
	glog.Info(args)
}

func Error(args ...interface{}) {
	glog.Info(args)
}
