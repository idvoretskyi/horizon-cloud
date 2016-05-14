package util

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func ValidateDomainName(domain string, fieldName string) error {
	// TODO: do more validation.
	if domain == "" {
		return fmt.Errorf("field `%s` empty", fieldName)
	}
	if len(domain) > 1024 {
		return fmt.Errorf("field `%s` too long (%v > %v)", fieldName, len(domain), 1024)
	}
	return nil
}

func ValidateProjectName(name string, fieldName string) error {
	if name == "" {
		return fmt.Errorf("field `%s` empty", fieldName)
	}
	if len(name) > 1024 {
		return fmt.Errorf("field `%s` too long (%v > %v)", fieldName, len(name), 1024)
	}
	return nil
}

func ValidateUserName(name string) error {
	if name == "" {
		return fmt.Errorf("empty user provided")
	}
	if len(name) > 100 {
		return fmt.Errorf("user `%v` too long (%v > %v)", name, len(name), 100)
	}
	for _, r := range name {
		if r == utf8.RuneError {
			return fmt.Errorf("user `%v` is not legal Unicode", name)
		}
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
