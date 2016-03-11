package util

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	maxNameLength = 24
	nameOverhead  = 4 // two character type, two `-`s.
	hashLength    = 8
	nameLength    = maxNameLength - (nameOverhead + hashLength)
)

var safeBytes = [256]bool{}

func init() {
	for i := 'a'; i <= 'z'; i++ {
		safeBytes[i] = true
	}
	for i := 'A'; i <= 'Z'; i++ {
		safeBytes[i] = true
	}
	for i := '0'; i <= '9'; i++ {
		safeBytes[i] = true
	}
	safeBytes['-'] = true
}

func TrueName(name string) string {
	rawHash := sha256.Sum256([]byte(name))
	hash := hex.EncodeToString(rawHash[:])[0:hashLength]

	safeName := strings.Map(func(r rune) rune {
		if r < 0 || r > 255 || !safeBytes[r] {
			return -1
		}
		return r
	}, name)
	if len(safeName) > nameLength {
		safeName = safeName[0:nameLength]
	}

	return safeName + "-" + hash
}
