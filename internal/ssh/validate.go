package ssh

import (
	"encoding/base64"

	cryptoSSH "golang.org/x/crypto/ssh"
)

func ValidKey(key string) bool {
	data, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return false
	}
	_, err = cryptoSSH.ParsePublicKey(data)
	return err == nil
}
