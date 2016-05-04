package api

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	TokenLifetime = time.Hour * 24

	maxClockSkew = time.Minute
)

type TokenData struct {
	Users []string
}

func SignToken(d *TokenData, key []byte) (string, error) {
	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims["u"] = d.Users
	t.Claims["issued"] = time.Now().Unix()
	t.Claims["maxage"] = TokenLifetime.Seconds()

	signed, err := t.SignedString(key)
	if err != nil {
		return "", err
	}

	return signed, nil
}

func VerifyToken(signed string, key []byte) (*TokenData, error) {
	t, err := jwt.Parse(signed, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid jwt algorithm")
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}

	if !t.Valid {
		return nil, errors.New("invalid jwt token")
	}

	checkTime := time.Now()

	issued, ok := t.Claims["issued"].(float64)
	if !ok {
		return nil, errors.New("no/bad issued field in token")
	}

	maxage, ok := t.Claims["maxage"].(float64)
	if !ok {
		return nil, errors.New("no/bad maxage field in token")
	}

	tokenAge := checkTime.Sub(time.Unix(int64(issued), 0))

	if tokenAge < -maxClockSkew {
		return nil, errors.New("token was created in the future")
	}
	if tokenAge.Seconds() > maxage {
		return nil, errors.New("token expired")
	}

	usersIf, ok := t.Claims["u"].([]interface{})
	if !ok {
		return nil, errors.New("no/bad u field in token")
	}

	users := make([]string, len(usersIf))
	for i := range usersIf {
		users[i], ok = usersIf[i].(string)
		if !ok {
			return nil, errors.New("bad u field in token")
		}
	}

	return &TokenData{
		Users: users,
	}, nil
}
