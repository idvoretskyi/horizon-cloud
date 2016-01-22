package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/db"
)

// RSI: find a way to figure out which fields were parsed and which
// were defaulted so that we can error if we get sent incomplete
// messages.

var rdb *db.DB

type validator interface {
	Validate() error
}

func decode(rw http.ResponseWriter, r io.Reader, body validator) bool {
	if err := json.NewDecoder(r).Decode(body); err != nil {
		writeJSONError(rw, http.StatusBadRequest, err)
		return false
	}
	if err := body.Validate(); err != nil {
		writeJSONError(rw, http.StatusBadRequest, err)
		return false
	}
	return true
}

type applyConfigRequest struct {
	Config api.Config
	Ready  <-chan struct{}
	DoIt   bool
}

func setConfig(rw http.ResponseWriter, req *http.Request) {
	var c api.Config
	if !decode(rw, req.Body, &c) {
		return
	}

	ready := make(chan struct{})
	defer close(ready)

	r := &applyConfigRequest{
		Config: c,
		Ready:  ready,
	}

	select {
	case applyConfigCh <- r:
	default:
		writeJSONError(rw, http.StatusInternalServerError,
			errors.New("too many pending configuration changes"))
		return
	}

	if err := rdb.SetConfig(&db.Config{Config: c}); err != nil {
		writeJSONError(rw, http.StatusInternalServerError, err)
		return
	}

	r.DoIt = true

	writeJSON(rw, http.StatusOK, c)
}

func getConfig(rw http.ResponseWriter, req *http.Request) {
	var gc api.GetConfigReq
	if !decode(rw, req.Body, &gc) {
		return
	}

	// RSI: don't let people read other people's configs.
	config, err := rdb.GetConfig(gc.Name)
	if err != nil {
		writeJSONError(rw, http.StatusInternalServerError, err)
		return
	}

	writeJSON(rw, http.StatusOK, api.GetConfigResp{
		Config: config.Config,
	})
}

func waitConfigApplied(rw http.ResponseWriter, req *http.Request) {
	var wca api.WaitConfigAppliedReq
	if !decode(rw, req.Body, &wca) {
		return
	}

	// RSI: access limitations a la getConfig

	returned := make(chan struct{})
	defer close(returned)

	var closeNotify <-chan bool
	if cnrw, ok := rw.(http.CloseNotifier); ok {
		closeNotify = cnrw.CloseNotify()
	}

	cancel := make(chan struct{})
	go func() {
		select {
		case <-returned:
			// do nothing
		case <-closeNotify:
			close(cancel)
		}
	}()

	config, err := rdb.WaitConfigApplied(wca.Name, wca.Version, cancel)
	if err != nil {
		writeJSONError(rw, http.StatusInternalServerError, err)
		return
	}

	writeJSON(rw, http.StatusOK, api.WaitConfigAppliedResp{
		Config: config.Config,
		// TODO: Target
	})
}

func main() {
	log.SetFlags(log.Lshortfile)

	var err error
	rdb, err = db.New()
	if err != nil {
		log.Fatal("unable to connect to RethinkDB: ", err)
	}

	// RSI: check for instances partially started

	http.HandleFunc("/v1/config/set", setConfig)
	http.HandleFunc("/v1/config/get", getConfig)
	http.HandleFunc("/v1/config/waitapplied", waitConfigApplied)
	log.Printf("Starting...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
