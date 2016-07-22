package main

import (
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
)

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
