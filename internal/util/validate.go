package util

import "fmt"

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
