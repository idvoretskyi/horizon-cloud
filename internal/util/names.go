package util

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	maxNameLength = 24
	nameOverhead  = 4 // two character type, two `-`s.
	hashLength    = 8
	nameLength    = maxNameLength - (nameOverhead + hashLength)
)

func TrueName(name string) string {
	rawHash := sha256.Sum256([]byte(name))
	hash := hex.EncodeToString(rawHash[:])[0:hashLength]
	truncName := name
	if len(truncName) > nameLength {
		truncName = truncName[0:nameLength]
	}
	return truncName + "-" + hash
}
