package api

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(rw http.ResponseWriter, code int, i interface{}) {
	rw.Header().Set("Content-Type", jsonMIMEType)
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(i)
}

func WriteJSONError(rw http.ResponseWriter, code int, err error) {
	WriteJSON(rw, code, map[string]string{"error": err.Error()})
}
