package util

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	maxKubeNameLength   = 24
	maxKubePrefixLength = 3

	maxTrueNameLength = maxKubeNameLength - maxKubePrefixLength
	hashLength        = 8
	textLength        = maxTrueNameLength - (hashLength + 1) // includes "-" separator
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
	safeName := strings.Map(func(r rune) rune {
		if r < 0 || r > 255 || !safeBytes[r] {
			return -1
		}
		return r
	}, name)
	if safeName == name && len(name) <= maxTrueNameLength {
		return name
	}

	rawHash := sha256.Sum256([]byte(name))
	hash := hex.EncodeToString(rawHash[:])[0:hashLength]

	if len(safeName) > textLength {
		safeName = safeName[0:textLength]
	}

	return safeName + "-" + hash
}
