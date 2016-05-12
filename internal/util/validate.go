package util

import (
	"fmt"
	"strings"
)

func ValidateDomainName(domain string, fieldName string) error {
	// RSI: do more validation.
	if domain == "" {
		return fmt.Errorf("field `%s` empty", fieldName)
	}
	return nil
}

func ValidateProjectName(name string, fieldName string) error {
	// RSI: more validation.
	if name == "" {
		return fmt.Errorf("field `%s` empty", fieldName)
	}
	return nil
}

func ValidateUserName(name string) error {
	// RSI: more validation.
	if name == "" {
		return fmt.Errorf("empty user provided")
	}
	return nil
}

// ReasonableToken returns true if the token could possibly be a JWT token.
func ReasonableToken(token string) bool {
	dots := 0
	for _, ch := range token {
		if ch == '.' {
			dots++
		} else if ch >= 'a' && ch <= 'z' ||
			ch >= 'A' && ch <= 'Z' ||
			ch >= '0' && ch <= '9' ||
			ch == '-' || ch == '_' || ch == '=' {
			// base64, ok
		} else {
			return false
		}
	}
	return dots == 2
}

// IsSafeRelPath returns true iff the path is relative, cannot escape the
// directory it starts in, and has no invalid characters.
//
// The path argument should be a slash-separated path, even on platforms that
// have different path separators.
func IsSafeRelPath(path string) bool {
	if strings.IndexByte(path, 0) != -1 {
		return false
	}
	if strings.IndexByte(path, '\n') != -1 {
		return false
	}
	if strings.IndexByte(path, '\r') != -1 {
		return false
	}
	if strings.HasPrefix(path, "/") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}
