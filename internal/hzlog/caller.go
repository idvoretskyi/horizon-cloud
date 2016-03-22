package hzlog

import (
	"os"
	"runtime"
	"strings"
)

type callerInfo struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

func getCallerInfo(calldepth int) callerInfo {
	_, file, line, ok := runtime.Caller(calldepth)
	if ok {
		// strip up to the second-to-last slash; so
		// "/home/ckastorff/src/github.com/rethinkdb/horizon-cloud/internal/hzlog/log.go"
		// becomes "hzlog/log.go"
		i := strings.LastIndexByte(file, os.PathSeparator)
		if i != -1 {
			i = strings.LastIndexByte(file[:i], os.PathSeparator)
		}
		if i != -1 {
			file = file[i+1:]
		}
	} else {
		file = "???"
		line = 0
	}
	return callerInfo{file, line}
}
