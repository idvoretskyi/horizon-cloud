package hzhttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rethinkdb/horizon-cloud/internal/hzhttp/hzlog"
)

func TestLogHTTPRequests(t *testing.T) {
	buf := &bytes.Buffer{}
	hzlog.SetOutput(buf)

	w := httptest.NewRecorder()
	innerH := HandlerFunc(func(c *Context, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, world!"))
	})
	outerH := LogHTTPRequests(innerH)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "1.2.3.4"

	outerH.ServeHTTPContext(NewContext(nil), w, req)

	var msg struct {
		HTTPRequest  miniReq  `json:"httprequest"`
		HTTPResponse miniResp `json:"httpresponse"`
	}
	err = json.NewDecoder(buf).Decode(&msg)
	if err != nil {
		t.Fatal(err)
	}

	wantReq := miniReq{
		Method:     req.Method,
		URL:        "/",
		Host:       "",
		RemoteAddr: "1.2.3.4",
	}
	if wantReq != msg.HTTPRequest {
		t.Errorf("Log returned minireq %#v, but wanted %#v",
			msg.HTTPRequest, wantReq)
	}
}
