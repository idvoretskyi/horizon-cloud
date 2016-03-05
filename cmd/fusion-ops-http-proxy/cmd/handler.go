package cmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/fusion-ops/cmd/fusion-ops-http-proxy/cache"
	"github.com/rethinkdb/fusion-ops/internal/api"
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
	spew.Dump(host)
	resp, err := h.conf.APIClient.GetByAlias(api.GetByAliasReq{
		SharedSecret: h.conf.APISecret,
		Alias:        host,
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// RSI: we may have to strip out the `:port` at the end in some cases.
	target, err := h.targetCache.Get(r.Host)
	if err != nil {
		http.Error(w, "Couldn't get proxy information for "+r.Host, http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	p, ok := h.proxies[target]
	if !ok {
		url, err := url.Parse(target)
		if err != nil {
			h.mu.Unlock()
			http.Error(w, "Proxy information invalid for "+r.Host, http.StatusInternalServerError)
			return
		}

		p = httputil.NewSingleHostReverseProxy(url)
		h.proxies[target] = p
	}
	h.mu.Unlock()

	p.ServeHTTP(w, r)
}
