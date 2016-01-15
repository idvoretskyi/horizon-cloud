package ssh

import (
	"io/ioutil"
	"testing"
)

func TestKnownHosts(t *testing.T) {
	kh, err := NewKnownHosts([]string{"line 1", "line 2"})
	if err != nil {
		t.Fatalf("Couldn't create new KnownHosts: %v", err)
	}

	data, err := ioutil.ReadFile(kh.Filename)
	if err != nil {
		t.Errorf("Couldn't read KnownHosts file from %v: %v", kh.Filename, err)
	}

	wanted := "line 1\nline 2\n"
	if string(data) != wanted {
		t.Errorf("%v has bad contents %#v, wanted %#v",
			kh.Filename, string(data), wanted)
	}

	err = kh.Close()
	if err != nil {
		t.Fatalf("Couldn't Close KnownHosts: %v", err)
	}
}
