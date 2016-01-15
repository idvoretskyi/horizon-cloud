package ssh

import "testing"

func TestShellEscape(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{"asdf", "asdf"},
		{"", "''"},
		{"'", "''\\'''"},
		{"Hello world", "'Hello world'"},
		{"--long-option", "--long-option"},
		{"Key=Value", "Key=Value"},
		{"Key=Value with Spaces", "'Key=Value with Spaces'"},
	}

	for _, test := range tests {
		out := shellEscape(test.Input)
		if out != test.Output {
			t.Errorf("shellEscape(%#v) = %#v, but wanted %#v",
				test.Input, out, test.Output)
		}
	}
}
