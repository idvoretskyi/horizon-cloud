package api

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
)

func RequireSecret(secret string, h hzhttp.Handler) hzhttp.Handler {
	secretBytes := []byte(secret)
	return hzhttp.HandlerFunc(func(c *hzhttp.Context, w http.ResponseWriter, r *http.Request) {
		gotValue := []byte(r.Header.Get(sharedSecretHeader))
		if subtle.ConstantTimeCompare(gotValue, secretBytes) == 0 {
			WriteJSONError(w, http.StatusForbidden, errors.New("shared secret missing or incorrect"))
			return
		}
		h.ServeHTTPContext(c, w, r)
	})
}
