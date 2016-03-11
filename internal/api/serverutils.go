package api

import (
	"crypto/subtle"
	"errors"
	"net/http"
)

func RequireSecret(secret string, h http.Handler) http.Handler {
	secretBytes := []byte(secret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotValue := []byte(r.Header.Get(sharedSecretHeader))
		if subtle.ConstantTimeCompare(gotValue, secretBytes) == 0 {
			WriteJSONError(w, http.StatusForbidden, errors.New("shared secret missing or incorrect"))
			return
		}
		h.ServeHTTP(w, r)
	})
}
