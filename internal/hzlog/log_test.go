package hzlog

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestLoggingWith(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)

	start := time.Now()

	blank := BlankLogger()
	blank.Output()
	blank.Log("msg")
	blank.Log("format %#v", "string")

	withA := blank.With(map[string]interface{}{"custom": "a"})
	withA.Output()
	withB := withA.With(map[string]interface{}{"custom": "b"})
	withB.Output()

	withA.Output()
	blank.Output()

	end := time.Now()

	type logMsg struct {
		Custom  string    `json:"custom"`
		Message string    `json:"message"`
		Time    time.Time `json:"time"`
	}

	ztime := time.Time{}

	wantMessages := []logMsg{
		{"", "", ztime},
		{"", "msg", ztime},
		{"", `format "string"`, ztime},
		{"a", "", ztime},
		{"b", "", ztime},
		{"a", "", ztime},
		{"", "", ztime},
	}

	at := 0
	decoder := json.NewDecoder(buf)
	for {
		var m logMsg
		err := decoder.Decode(&m)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if at >= len(wantMessages) {
			t.Errorf("Got extra log structure %#v", m)
			continue
		}

		want := wantMessages[at]
		if want.Custom != m.Custom || want.Message != m.Message {
			t.Errorf("Bad values in log index %v, have %#v but wanted %#v",
				at, m, want)
		}
		if m.Time.Before(start) || m.Time.After(end) {
			t.Errorf("Log index %v time %v is out of range %v - %v",
				at, m.Time, start, end)
		}
		at++
	}
}

func TestLoggingCallerOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)

	BlankLogger().Output()

	var logMsg struct {
		Source callerInfo `json:"source"`
	}
	err := json.NewDecoder(buf).Decode(&logMsg)
	if err != nil {
		t.Fatal(err)
	}

	if logMsg.Source.File != "hzlog/log_test.go" || logMsg.Source.Line <= 0 {
		t.Errorf("Bad caller information in log %#v", logMsg)
	}
}

func TestLoggingCallerLog(t *testing.T) {
	buf := &bytes.Buffer{}
	SetOutput(buf)

	BlankLogger().Log("message")

	var logMsg struct {
		Source callerInfo `json:"source"`
	}
	err := json.NewDecoder(buf).Decode(&logMsg)
	if err != nil {
		t.Fatal(err)
	}

	if logMsg.Source.File != "hzlog/log_test.go" || logMsg.Source.Line <= 0 {
		t.Errorf("Bad caller information in log %#v", logMsg)
	}
}
