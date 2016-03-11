package cmd

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/horizon-cloud/cmd/horizon-cloud-http-proxy/cache"
	"github.com/rethinkdb/horizon-cloud/internal/api"
)

type Handler struct {
	conf        *config
	targetCache *cache.Cache

	mu      sync.Mutex
	proxies map[string]*httputil.ReverseProxy
}

func NewHandler(conf *config) *Handler {
	h := &Handler{
		conf: conf,
		// RSI: remove ReverseProxies from this map if no requests are
		// coming in for them.
		proxies: make(map[string]*httputil.ReverseProxy, 128),
	}

	h.targetCache = cache.New(h.lookupTargetForHost)

	return h
}

func (h *Handler) lookupTargetForHost(host string) (string, error) {
	spew.Dump(host)
	resp, err := h.conf.APIClient.GetByAlias(api.GetByAliasReq{
		Alias: host,
	})
	// RSI: log error.
	spew.Dump(resp)
	spew.Dump(err)
	if err != nil {
		return "", err
	}
	if resp.Project == nil {
		// RSI: consider passing this on so users can distinguish between
		// not having registered an alias and other errors in production.
		return "", fmt.Errorf("no alias registered for `%s`", host)
	}
	return resp.Project.HTTPAddress, nil
}

func maybeCloseWrite(c net.Conn) {
	cw, ok := c.(interface {
		CloseWrite() error
	})
	if ok {
		cw.CloseWrite()
	} else {
		c.Close()
	}
}

func maybeCloseRead(c net.Conn) {
	cw, ok := c.(interface {
		CloseRead() error
	})
	if ok {
		cw.CloseRead()
	} else {
		c.Close()
	}
}

func websocketProxy(target string, w http.ResponseWriter, r *http.Request) {
	d, err := net.Dial("tcp", target)
	if err != nil {
		http.Error(w, "Error contacting backend server.", 500)
		log.Printf("Error dialing websocket backend %s: %v", target, err)
		return
	}
	defer d.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "internal error", 500)
		// RSI: log a serious error.
		return
	}
	nc, buf, err := hj.Hijack()
	if err != nil {
		http.Error(w, "internal error", 500)
		// RSI: log a serious error.
		return
	}
	defer nc.Close()

	err = r.Write(d)
	if err != nil {
		log.Printf("Error copying request to target: %v", err)
		return
	}

	done := make(chan struct{})
	go func() {
		io.Copy(d, buf)
		maybeCloseWrite(d)
		maybeCloseRead(nc)
		close(done)
	}()
	io.Copy(nc, d)
	maybeCloseWrite(nc)
	maybeCloseRead(d)
	<-done
}

func isWebsocket(req *http.Request) bool {
	if strings.ToLower(req.Header.Get("Connection")) == "upgrade" {
		for _, uhdr := range req.Header["Upgrade"] {
			if strings.ToLower(uhdr) == "websocket" {
				return true
			}
		}
	}
	return false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// RSI: we may have to strip out the `:port` at the end in some cases.
	target, err := h.targetCache.Get(r.Host)
	if err != nil {
		http.Error(w, "Couldn't get proxy information for "+r.Host,
			http.StatusInternalServerError)
		return
	}

	if isWebsocket(r) {
		log.Printf("%p serving websocket: %s", r, r.URL.Path)
		websocketProxy(target, w, r)
		return
	}

	log.Printf("%p serving http: %s", r, r.URL.Path)
	h.mu.Lock()
	p, ok := h.proxies[target]
	if !ok {
		url, err := url.Parse("http://" + target)
		if err != nil {
			h.mu.Unlock()
			http.Error(w, "Proxy information invalid for "+r.Host,
				http.StatusInternalServerError)
			return
		}

		p = httputil.NewSingleHostReverseProxy(url)
		h.proxies[target] = p
	}
	h.mu.Unlock()

	p.ServeHTTP(w, r)
}
