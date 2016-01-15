package ssh

import "strings"

var safeShellRunes = map[rune]struct{}{}

func init() {
	for _, r := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_.,/=" {
		safeShellRunes[r] = struct{}{}
	}
}

// ShellEscape returns a string that can be passed as an argument to a command
// in a shell that represents the same bytes as the string passed to it.
//
// If s contains a null byte, which there is no escape for, ShellEscape panics.
func ShellEscape(s string) string {
	if strings.ContainsRune(s, '\x00') {
		panic("cannot shell-escape a string containing nulls")
	}

	if s == "" {
		return "''"
	}

	isSafe := true
	for _, r := range s {
		if _, ok := safeShellRunes[r]; !ok {
			isSafe = false
			break
		}
	}

	if isSafe {
		return s
	}

	return "'" + strings.Replace(s, "'", "'\\''", -1) + "'"
}

// ShellEscapeJoin returns the result of calling ShellEscape on each element of
// the given array, joined by spaces.
func ShellEscapeJoin(ss []string) string {
	esc := make([]string, len(ss))
	for i, s := range ss {
		esc[i] = ShellEscape(s)
	}
	return strings.Join(esc, " ")
}
