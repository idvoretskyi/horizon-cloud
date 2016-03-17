package main

import (
	"encoding/binary"
	"errors"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

var (
	errPairTooShort = errors.New("pair too short")
	errPairInvalid  = errors.New("pair invalid")
)

func decodeLengthPrefixedKV(b []byte) ([]byte, []byte, error) {
	// RSI(sec): go-fuzz this for unintended panics

	if len(b) < 4 {
		return nil, nil, errPairTooShort
	}

	keyLength := int(binary.BigEndian.Uint32(b[:4]))
	b = b[4:]

	if keyLength < 0 {
		return nil, nil, errPairTooShort
	}
	if len(b) < keyLength {
		return nil, nil, errPairTooShort
	}

	key := b[:keyLength]
	b = b[keyLength:]

	if len(b) < 4 {
		return nil, nil, errPairTooShort
	}

	valueLength := int(binary.BigEndian.Uint32(b[:4]))
	b = b[4:]

	if valueLength < 0 {
		return nil, nil, errPairInvalid
	}
	if len(b) < valueLength {
		return nil, nil, errPairTooShort
	}

	value := b[:valueLength]
	b = b[valueLength:]

	if len(b) != 0 {
		return nil, nil, errPairInvalid
	}

	return key, value, nil
}

func loadPrivateKey(path string) (ssh.Signer, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(bytes)
}
