package util

import "testing"

func TestNames(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{"", "-e3b0c442"},
		{"foo", "foo-2c26b46b"},
		{"horizon-cloud", "horizon-clou-bdc7f4a5"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"ffffffffffff-d758d73f"},
		{"space   the final frontier", "spacethefina-4dd46748"},
	}

	for _, test := range tests {
		out := TrueName(test.Input)
		if out != test.Output {
			t.Errorf("TrueName(%#v) = %#v, expected %#v",
				test.Input, out, test.Output)
		}
	}
}
