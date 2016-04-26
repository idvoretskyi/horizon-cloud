package hzlog

import (
	"bytes"
	"io"
	"sync"
)

type writerLogger struct {
	log *Logger

	mu  sync.Mutex
	buf bytes.Buffer
}

// WriterLogger returns a new io.Writer, to which when entire lines are
// written, writes a log message to the Logger given.
func WriterLogger(log *Logger) io.Writer {
	wl := &writerLogger{
		log: log,
	}
	return wl
}

func (wl *writerLogger) Write(p []byte) (int, error) {
	wl.mu.Lock()
	_, _ = wl.buf.Write(p)

	for {
		hasNewline := false
		for _, b := range wl.buf.Bytes() {
			if b == '\n' {
				hasNewline = true
				break
			}
		}
		if !hasNewline {
			break
		}

		line, _ := wl.buf.ReadBytes('\n')
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		wl.log.Info("%s", string(line))
	}

	wl.mu.Unlock()
	return len(p), nil
}
