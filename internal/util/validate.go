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
