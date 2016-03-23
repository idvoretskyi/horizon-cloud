package hzlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	infoLevelMap      = map[string]interface{}{"level": "info"}
	userErrorLevelMap = map[string]interface{}{"level": "user_error"}
	errorLevelMap     = map[string]interface{}{"level": "error"}
)

func init() {
	SetOutput(os.Stdout)
}

var (
	outputWriter  io.Writer
	outputEncoder *json.Encoder
	outputMutex   sync.Mutex
)

func SetOutput(w io.Writer) {
	outputMutex.Lock()
	outputWriter = w
	outputEncoder = json.NewEncoder(w)
	outputMutex.Unlock()
}

type Logger struct {
	partial map[string]interface{}
	parent  *Logger
}

func BlankLogger() *Logger {
	return &Logger{nil, nil}
}

func (c *Logger) OutputDepth(calldepth int) {
	out := make(map[string]interface{}, c.entryCount()+2)
	for tc := c; tc != nil; tc = tc.parent {
		if tc.partial != nil {
			for k, v := range tc.partial {
				if _, ok := out[k]; !ok {
					out[k] = v
				}
			}
		}
	}

	out["time"] = time.Now().UTC()
	out["source"] = getCallerInfo(calldepth + 1)

	outputMutex.Lock()
	outputEncoder.Encode(out)
	outputMutex.Unlock()
}

func (c *Logger) Output() {
	c.OutputDepth(2)
}

func (c *Logger) LogDepth(calldepth int, format string, args ...interface{}) {
	c.With(map[string]interface{}{
		"message": fmt.Sprintf(format, args...),
	}).OutputDepth(calldepth + 1)
}

func (c *Logger) Log(format string, args ...interface{}) {
	c.LogDepth(2, format, args...)
}

func (c *Logger) ErrorDepth(calldepth int, format string, args ...interface{}) {
	c.With(errorLevelMap).LogDepth(calldepth+1, format, args...)
}

func (c *Logger) Error(format string, args ...interface{}) {
	c.ErrorDepth(2, format, args...)
}

func (c *Logger) InfoDepth(calldepth int, format string, args ...interface{}) {
	c.With(infoLevelMap).LogDepth(calldepth+1, format, args...)
}

func (c *Logger) Info(format string, args ...interface{}) {
	c.InfoDepth(2, format, args...)
}

func (c *Logger) UserErrorDepth(calldepth int, format string, args ...interface{}) {
	c.With(userErrorLevelMap).LogDepth(calldepth+1, format, args...)
}

func (c *Logger) UserError(format string, args ...interface{}) {
	c.UserErrorDepth(2, format, args...)
}

func (c *Logger) With(m map[string]interface{}) *Logger {
	return &Logger{m, c}
}

func (c *Logger) entryCount() int {
	out := len(c.partial)
	if c.parent != nil {
		out += c.parent.entryCount()
	}
	return out
}
