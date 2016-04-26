package util

import "testing"

func TestNames(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{"", ""},
		{"foo", "foo"},
		{"horizon-cloud", "horizon-cloud"},
		{"horizon-clou-bdc7f4a5", "horizon-clou-bdc7f4a5"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"ffffffffffff-d758d73f"},
		{"space   the final frontier", "spacethefina-4dd46748"},
		{"z z", "zz-79b652ee"},
	}

	for _, test := range tests {
		out := TrueName(test.Input)
		if out != test.Output {
			t.Errorf("TrueName(%#v) = %#v, expected %#v",
				test.Input, out, test.Output)
		}
	}
}
