package util

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	maxKubeObjectLength = 24
	maxKubePrefixLength = 3

	kubeNameLength = maxKubeObjectLength - maxKubePrefixLength
)

func KubeName(ownerName string, projectName string) string {
	compositeName := ownerName + "/" + projectName
	rawHash := sha256.Sum256([]byte(compositeName))
	return hex.EncodeToString(rawHash[:])[0:kubeNameLength]
}
