package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/rethinkdb/fusion-ops/api"
	"github.com/rethinkdb/fusion-ops/db"
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

func create(rw http.ResponseWriter, req *http.Request) {
	var c api.Config
	if !decode(rw, req.Body, &c) {
		return
	}

	if err := rdb.CreateConfig(&c); err != nil {
		writeJSONError(rw, http.StatusInternalServerError, err)
		return
	}

	// Do AWS configuration
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
		Config: *config,
	})
}

func main() {
	var err error
	rdb, err = db.New()
	if err != nil {
		log.Fatal("unable to connect to RethinkDB: ", err)
	}
	http.HandleFunc("/v1/create", create)
	http.HandleFunc("/v1/reconfig", create)
	http.HandleFunc("/v1/getConfig", getConfig)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
