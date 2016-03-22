package hzlog

import (
	"bufio"
	"io"
	"sync"

	"gopkg.in/tomb.v2"
)

type writerLogger struct {
	log *Logger
	t   tomb.Tomb

	pwMu sync.Mutex
	pw   *io.PipeWriter

	// pr and buf are owned by wl.readLoop()
	pr  *io.PipeReader
	buf *bufio.Reader
}

// WriterLogger returns a new io.WriteCloser, to which when entire lines are
// written, writes a log message to the Logger given.
func WriterLogger(log *Logger) io.WriteCloser {
	pr, pw := io.Pipe()
	buf := bufio.NewReader(pr)
	wl := &writerLogger{
		log: log,
		pr:  pr,
		pw:  pw,
		buf: buf,
	}
	wl.t.Go(func() error {
		wl.t.Go(wl.readLoop)
		wl.t.Go(func() error {
			// This routine ensures that calls to wl.Write will fail (instead of
			// hang) if the readLoop crashes or wl.Close is called.
			<-wl.t.Dying()
			return wl.pw.Close()
		})
		return nil
	})
	return wl
}

func (wl *writerLogger) Write(p []byte) (int, error) {
	wl.pwMu.Lock()
	n, err := wl.pw.Write(p)
	wl.pwMu.Unlock()
	return n, err
}

func (wl *writerLogger) Close() error {
	wl.pwMu.Lock()
	wl.t.Kill(nil)
	wl.pwMu.Unlock()
	return wl.t.Wait()
}

func (wl *writerLogger) readLoop() error {
	for {
		line, err := wl.buf.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			// Someone sent a really, really long line to us; we'll just split
			// it into multiple messages, since we don't want to buffer
			// infinitely.
			err = nil
		}
		if err != nil {
			return err
		}

		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		wl.log.Log("%s", string(line))
	}
}
