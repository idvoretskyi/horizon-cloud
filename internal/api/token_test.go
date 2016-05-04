package api

import (
	"reflect"
	"testing"

	"github.com/dgrijalva/jwt-go"
)

func TestTokenVerifesValid(t *testing.T) {
	data := &TokenData{Users: []string{"the thinker"}}

	signed, err := SignToken(data, []byte("key"))
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := VerifyToken(signed, []byte("key"))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(decoded, data) {
		t.Fatalf("data was %v, wanted %v", decoded, data)
	}
}

func TestTokenRejectsBadKey(t *testing.T) {
	signed, err := SignToken(&TokenData{Users: []string{"u"}}, []byte("key"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = VerifyToken(signed, []byte("wrong key"))
	if err == nil {
		t.Fatal("Got no error verifying a token signed with a different key")
	}
}

func TestTokenRejectsNoneAlg(t *testing.T) {
	tok := jwt.New(jwt.SigningMethodNone)
	tok.Claims["User"] = "someone"
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = VerifyToken(signed, []byte("no key"))
	if err == nil {
		t.Fatal("Got no error verifying a token with the 'none' signing method")
	}
}
