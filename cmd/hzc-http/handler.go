package main

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
	"time"

	"github.com/encryptio/go-meetup"
	"github.com/rethinkdb/horizon-cloud/internal/api"
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
	resp, err := h.conf.APIClient.GetProjectByDomain(api.GetProjectByDomainReq{
		Domain: host,
	})
	// RSI: log error.
	if err != nil {
		return "", err
	}
	if resp.Project == nil {
		return "", &NoHostMappingError{host}
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

func isTLSOnlyHost(host string) bool {
	return isHSTSHost(host)
}

func isHSTSHost(host string) bool {
	// TODO: make this more dynamic
	if strings.HasSuffix(host, ".hzc.io") {
		return true
	}
	return false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// RSI: we may have to strip out the `:port` at the end in some cases.
	target, err := h.getCachedTarget(r.Host)
	if err != nil {
		if _, ok := err.(*NoHostMappingError); ok {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "Couldn't get proxy information for "+r.Host,
			http.StatusInternalServerError)
		return
	}

	if isTLSOnlyHost(r.Host) && r.TLS == nil {
		httpsURL := *r.URL
		httpsURL.Scheme = "https"
		httpsURL.Host = r.Host
		w.Header().Set("Location", httpsURL.String())
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	if isHSTSHost(r.Host) {
		w.Header().Set("Strict-Transport-Security", "max-age=10886400; includeSubDomains")
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
