package hzhttp

import (
	"math/rand"
	"net/http"
	"time"
)

// A Handler responds to an HTTP request. See http.Handler for more details.
type Handler interface {
	ServeHTTPContext(c *Context, w http.ResponseWriter, r *http.Request)
}

// HandlerFunc wraps a plain function as a Handler.
type HandlerFunc func(c *Context, w http.ResponseWriter, r *http.Request)

// ServeHTTPContext implements Handler.
func (f HandlerFunc) ServeHTTPContext(c *Context, w http.ResponseWriter, r *http.Request) {
	f(c, w, r)
}

// BaseContext returns an http.Handler (NB: not an hzhttp.Handler) that always
// passes the given context to its Handler.
func BaseContext(c *Context, h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTPContext(c, w, r)
	})
}

func randomString() string {
	b := make([]byte, 10)
	for i := range b {
		b[i] = byte(rand.Intn('z'-'a'+1) + 'a')
	}
	return string(b)
}

type miniReq struct {
	Method     string `json:"method"`
	URL        string `json:"url"`
	Host       string `json:"host"`
	RemoteAddr string `json:"remoteaddr"`
	RequestID  string `json:"requestid"`
}

type miniResp struct {
	Duration float64           `json:"duration"`
	Read     transferStatsWire `json:"read"`
	Write    transferStatsWire `json:"write"`
	Status   int               `json:"status"`
}

// LogHTTPRequests logs some basic information about the HTTP request and
// response, and also adds the `httprequest` log field to the Context it passes
// on.
func LogHTTPRequests(h Handler) Handler {
	return HandlerFunc(func(c *Context, w http.ResponseWriter, r *http.Request) {
		// TODO: load request ID from header, save in outgoing requests
		requestID := randomString()

		c = c.WithLog(map[string]interface{}{
			"httprequest": miniReq{
				Method:     r.Method,
				URL:        r.URL.String(),
				Host:       r.Host,
				RemoteAddr: r.RemoteAddr,
				RequestID:  requestID,
			},
		})

		started := time.Now()
		var rws responseWriterState
		var body *readTracker
		if r.Body != nil {
			body = &readTracker{ReadCloser: r.Body}
			r.Body = body
		}
		h.ServeHTTPContext(c, wrapResponseWriter(w, &rws), r)
		duration := time.Now().Sub(started)

		respStats := miniResp{
			Duration: duration.Seconds(),
			Write:    rws.Transfer.Wire(),
			Status:   rws.Status,
		}

		if body != nil {
			respStats.Read = body.Stats.Wire()
		}

		c = c.WithLog(map[string]interface{}{"httpresponse": respStats})
		c.EmptyLog()
	})
}
