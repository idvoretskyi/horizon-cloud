package kube

import (
	"fmt"
	"testing"
)

func TestCompositeErr(t *testing.T) {
	errFoo := fmt.Errorf("foo")
	errBar := fmt.Errorf("bar")
	errBaz := fmt.Errorf("baz")

	tests := []struct {
		Error         error
		ExpectedError string
	}{
		{compositeErr(), ""},
		{compositeErr(errFoo), "foo"},
		{compositeErr(errFoo, errBar), "composite error: [foo, bar]"},
		{compositeErr(errFoo, errBar, errBaz), "composite error: [foo, bar, baz]"},
		{compositeErr(nil), ""},
		{compositeErr(nil, nil, nil), ""},
		{compositeErr(errFoo, nil), "foo"},
		{compositeErr(errFoo, nil, errBar), "composite error: [foo, bar]"},
		{compositeErr(nil, errFoo, errBar, nil, errBaz), "composite error: [foo, bar, baz]"},
	}
	for idx, test := range tests {
		if test.Error == nil {
			if test.ExpectedError != "" {
				t.Errorf("error (%v): nil != %#v", idx, test.ExpectedError)
			}
		} else {
			if test.Error.Error() != test.ExpectedError {
				t.Errorf("error (%v): %#v != %#v", idx, test.Error.Error(), test.ExpectedError)
			}
		}
	}
}
