package main

import (
	"encoding/json"
	"net/http"

	"github.com/rethinkdb/fusion-ops/api"
)

func writeJSON(rw http.ResponseWriter, code int, i interface{}) {
	// RSI: log errors from writeJSON.
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(i)
}

func writeJSONError(rw http.ResponseWriter, code int, err error) {
	writeJSON(rw, code, api.Resp{
		Success: false,
		Error:   err.Error(),
	})
}

func writeJSONResp(rw http.ResponseWriter, code int, i interface{}) {
	r := api.Resp{Success: true}
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	r.Content = json.RawMessage(b)
	writeJSON(rw, code, r)
}
