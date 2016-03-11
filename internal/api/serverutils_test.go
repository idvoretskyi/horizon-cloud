package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireSecret(t *testing.T) {
	const secret = "hunter2"

	executions := 0
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		executions++
	})

	secureH := RequireSecret(secret, h)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	secureH.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("Unauthenticated request returned response code %v", recorder.Code)
	}
	if executions != 0 {
		t.Errorf("Unauthenticated request still executed")
	}

	recorder = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(sharedSecretHeader, secret)
	secureH.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Authenticated request returned response code %v", recorder.Code)
	}
	if executions != 1 {
		t.Errorf("Authenticated request did not execute")
	}
}
