package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/rethinkdb/fusion-ops/cmd/fusion-ops-http-proxy/cache"
)

type Handler struct {
	conf        *config
	targetCache *cache.Cache

	mu      sync.Mutex
	proxies map[string]*httputil.ReverseProxy
}

func NewHandler(conf *config) *Handler {
	h := &Handler{
		conf:    conf,
		proxies: make(map[string]*httputil.ReverseProxy, 128),
	}

	h.targetCache = cache.New(h.lookupTargetForHost)

	return h
}

func (h *Handler) lookupTargetForHost(host string) (string, error) {
	return "http://rethinkdb.com:80", nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := h.targetCache.Get(r.URL.Host)
	if err != nil {
		http.Error(w, "Couldn't get proxy information for "+r.URL.Host, http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	p, ok := h.proxies[target]
	if !ok {
		url, err := url.Parse(target)
		if err != nil {
			h.mu.Unlock()
			http.Error(w, "Proxy information invalid for "+r.URL.Host, http.StatusInternalServerError)
			return
		}

		p = httputil.NewSingleHostReverseProxy(url)
		h.proxies[target] = p
	}
	h.mu.Unlock()

	p.ServeHTTP(w, r)
}
