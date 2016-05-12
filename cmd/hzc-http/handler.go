package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/encryptio/go-meetup"
	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
)

type NoHostMappingError struct {
	Host string
}

func (e *NoHostMappingError) Error() string {
	return fmt.Sprintf("no host mapping exists for %v", e.Host)
}

type Handler struct {
	conf        *config
	targetCache *meetup.Cache

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

	h.targetCache = meetup.New(meetup.Options{
		Get: func(host string) (interface{}, error) {
			return h.lookupTargetForHost(host)
		},

		Concurrency:   20,
		ErrorAge:      time.Second,
		ExpireAge:     time.Hour,
		RevalidateAge: time.Minute,
	})

	return h
}

func (h *Handler) getCachedTarget(host string) (string, error) {
	v, err := h.targetCache.Get(host)
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (h *Handler) lookupTargetForHost(host string) (string, error) {
	resp, err := h.conf.APIClient.GetProjectAddrByDomain(api.GetProjectAddrByDomainReq{
		Domain: host,
	})
	// RSI: log error.
	if err != nil {
		return "", err
	}
	if resp.ProjectAddr == nil {
		return "", &NoHostMappingError{host}
	}
	return resp.ProjectAddr.HTTPAddr, nil
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

func websocketProxy(
	target string, ctx *hzhttp.Context, w http.ResponseWriter, r *http.Request) {

	d, err := net.Dial("tcp", target)
	if err != nil {
		http.Error(w, "Error contacting backend server.", 500)
		ctx.Error("Error dialing websocket backend %s: %v", target, err)
		return
	}
	defer d.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		ctx.Error("ResponseWriter was not a hijacker")
		http.Error(w, "internal error", 500)
		return
	}
	nc, buf, err := hj.Hijack()
	if err != nil {
		ctx.Error("ResponseWriter failed to hijack: %v", err)
		http.Error(w, "internal error", 500)
		return
	}
	defer nc.Close()

	err = r.Write(d)
	if err != nil {
		ctx.Info("Failed to write request to backend: %v", err)
		return
	}

	done := make(chan struct{})
	go func() {
		_, err := io.Copy(d, buf)
		if err != nil && err != io.EOF {
			ctx.Info("Failed to copy to backend: %v", err)
		}
		maybeCloseWrite(d)
		maybeCloseRead(nc)
		close(done)
	}()
	_, err = io.Copy(nc, d)
	if err != nil && err != io.EOF {
		ctx.Info("Failed to copy from client: %v", err)
	}
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

func isTLSOnlyHost(host string) bool {
	return isHSTSHost(host)
}

func isHSTSHost(host string) bool {
	// RSI: make this more dynamic
	if strings.HasSuffix(host, ".hzc.io") {
		return true
	}
	return false
}

func (h *Handler) ServeHTTPContext(
	ctx *hzhttp.Context, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	r.URL.Path = "/horizon"
	if path == "" { // We have to check this so the slice is legal.
		http.Error(w, "no host specified", http.StatusNotFound)
		return
	}
	host := path[1:]
	target, err := h.getCachedTarget(host)
	if err != nil {
		if _, ok := err.(*NoHostMappingError); ok {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		ctx.Error("Couldn't get proxy information for %s: %v", host, err)
		http.Error(w, "Couldn't get proxy information for "+host,
			http.StatusInternalServerError)
		return
	}

	if isTLSOnlyHost(host) && r.TLS == nil {
		httpsURL := *r.URL
		httpsURL.Scheme = "https"
		httpsURL.Host = host
		w.Header().Set("Location", httpsURL.String())
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	if isHSTSHost(host) {
		w.Header().Set("Strict-Transport-Security", "max-age=10886400; includeSubDomains")
	}

	if isWebsocket(r) {
		ctx.Info("serving as websocket")
		websocketProxy(target, ctx, w, r)
		return
	}

	h.mu.Lock()
	p, ok := h.proxies[target]
	if !ok {
		url, err := url.Parse("http://" + target)
		if err != nil {
			h.mu.Unlock()
			ctx.Error("Proxy information invalid for %s: %v", host, err)
			http.Error(w, "Proxy information invalid for "+host,
				http.StatusInternalServerError)
			return
		}

		p = httputil.NewSingleHostReverseProxy(url)
		h.proxies[target] = p
	}
	h.mu.Unlock()

	p.ServeHTTP(w, r)
}
