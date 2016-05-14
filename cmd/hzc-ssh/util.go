package main

import (
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

func loadPrivateKey(path string) (ssh.Signer, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(bytes)
}
