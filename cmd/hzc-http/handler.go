package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/encryptio/go-meetup"
	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/types"
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
	ctx         *hzhttp.Context
	proxy       *httputil.ReverseProxy
}

func NewHandler(conf *config, ctx *hzhttp.Context) *Handler {
	h := &Handler{
		conf: conf,
		ctx:  ctx,
		proxy: &httputil.ReverseProxy{
			Director: func(r *http.Request) {},
		},
		targetCache: meetup.New(meetup.Options{
			Get: func(host string) (interface{}, error) {
				resp, err := conf.APIClient.GetProjectAddrByDomain(api.GetProjectAddrByDomainReq{
					Domain: host,
				})
				if err != nil {
					ctx.Error("API server gave no response for `%v` (%v)", host, err)
					return nil, err
				}
				if resp.ProjectAddr == nil {
					return nil, &NoHostMappingError{host}
				}
				return resp.ProjectAddr, nil
			},
			Concurrency:   20,
			ErrorAge:      time.Second,
			RevalidateAge: time.Minute,
			MaxSize:       100000, // very roughly 64MB of stuff (TODO: more precisely derive this)
		}),
	}

	return h
}

func (h *Handler) getCachedTarget(host string) (*types.ProjectAddr, error) {
	v, err := h.targetCache.Get(host)
	if err != nil {
		return nil, err
	}
	return v.(*types.ProjectAddr), nil
}

func (h *Handler) ServeHTTPContext(
	ctx *hzhttp.Context, w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	slashIndex := strings.IndexByte(path, '/')
	if slashIndex == -1 {
		http.Error(w, "malformed path", http.StatusNotFound)
		return
	}
	host := path[:slashIndex]
	r.URL.Path = path[slashIndex:]
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

	// Websocket requests proxied with TCP to the horizon pod
	if isWebsocket(r) {
		ctx.Info("serving as websocket")
		websocketProxy(target.HTTPAddr, ctx, w, r)
		return
	}

	// /horizon/ requests proxied with HTTP to the horizon pod
	if strings.HasPrefix(r.URL.Path, "/horizon/") {
		r.URL.Scheme = "http"
		r.URL.Host = target.HTTPAddr
		h.proxy.ServeHTTP(w, r)
		return
	}

	// All other requests proxied with HTTP to GCS
	r.URL.Scheme = "https"
	r.URL.Host = "storage.googleapis.com"
	r.URL.Path = target.GCSPrefix + strings.TrimPrefix(r.URL.Path, "/")
	if strings.HasSuffix(r.URL.Path, "/") {
		r.URL.Path += "index.html"
	}
	h.proxy.ServeHTTP(w, r)
}
