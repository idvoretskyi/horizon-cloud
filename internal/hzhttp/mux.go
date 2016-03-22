package hzhttp

import (
	"net/http"
	"sync"
)

var _ Handler = &Muxer{}

// A Muxer multiplexes between multiple Handlers based on the request paths.
//
// If a request comes in that does not match any registered Handler, then the
// Muxer responds with a not found error.
type Muxer struct {
	mu    sync.RWMutex
	paths map[string]Handler
}

// NewMuxer returns a new, empty Muxer.
func NewMuxer() *Muxer {
	return &Muxer{
		paths: make(map[string]Handler, 8),
	}
}

// RegisterPath registers a path with a given handler. The path is matched
// precisely (no prefix logic is done.)
func (mux *Muxer) RegisterPath(path string, h Handler) {
	mux.mu.Lock()
	mux.paths[path] = h
	mux.mu.Unlock()
}

// ServeHTTPContext implements Handler.
func (mux *Muxer) ServeHTTPContext(c *Context, w http.ResponseWriter, r *http.Request) {
	mux.mu.RLock()
	h := mux.paths[r.URL.Path]
	mux.mu.RUnlock()

	if h == nil {
		http.NotFound(w, r)
		return
	}

	h.ServeHTTPContext(c, w, r)
}
