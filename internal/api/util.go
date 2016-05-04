package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Resp struct {
	Success bool
	Error   string           `json:",omitempty"`
	Content *json.RawMessage `json:",omitempty"`
}

func writeJSON(rw http.ResponseWriter, code int, i interface{}) {
	// RSI: log errors from writeJSON.
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(i)
}

func WriteJSONError(rw http.ResponseWriter, code int, err error) {
	writeJSON(rw, code, Resp{
		Success: false,
		Error:   err.Error(),
	})
}

func WriteJSONResp(rw http.ResponseWriter, code int, i interface{}) {
	r := Resp{Success: true}
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	rm := json.RawMessage(b)
	r.Content = &rm
	writeJSON(rw, code, r)
}

func ReadJSONResp(hr *http.Response, i interface{}) error {
	var resp Resp
	err := json.NewDecoder(hr.Body).Decode(&resp)
	if err != nil {
		return err
	}
	if !resp.Success || hr.StatusCode != http.StatusOK {
		return fmt.Errorf("server error (%s): %s", hr.Status, resp.Error)
	}
	json.Unmarshal(*resp.Content, i)
	return nil
}
