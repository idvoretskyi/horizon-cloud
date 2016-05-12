package util

import (
	"crypto/sha256"
	"encoding/hex"
)

func BytesToHash(b []byte) string {
	shaBytes := sha256.Sum256(b)
	return hex.EncodeToString(shaBytes[:])
}
