package hzlog

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestWriterLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)

	wl := WriterLogger(BlankLogger())

	_, err := wl.Write([]byte("a\nb\n"))
	if err != nil {
		t.Error(err)
	}

	expect := []string{"a", "b"}
	at := 0

	decoder := json.NewDecoder(buf)
	for {
		var logMsg struct {
			Message string `json:"message"`
		}
		err := decoder.Decode(&logMsg)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if at >= len(expect) {
			t.Error("Got extraneous log message %v", logMsg)
			continue
		}

		if logMsg.Message != expect[at] {
			t.Errorf("Got message %#v but wanted %#v", logMsg.Message, expect[at])
		}
		at++
	}
}
