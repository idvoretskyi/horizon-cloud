package ssh

import (
	"fmt"
	"io/ioutil"
	"os"
)

// A KnownHosts object represents a known_hosts file stored in a temporary file
// on disk.
type KnownHosts struct {
	Lines    []string
	Filename string
}

// NewKnownHosts creates a new known_hosts file with the given lines in it.
func NewKnownHosts(lines []string) (*KnownHosts, error) {
	kh := &KnownHosts{Lines: lines}
	err := kh.open()
	if err != nil {
		return nil, err
	}

	return kh, nil
}

func (kh *KnownHosts) open() error {
	f, err := ioutil.TempFile("", "fusion-known-hosts")
	if err != nil {
		return err
	}

	kh.Filename = f.Name()

	for _, s := range kh.Lines {
		fmt.Fprintf(f, "%s\n", s)
	}

	return f.Close()
}

// Close cleans up the temporary file used by the KnownHosts object.
func (kh *KnownHosts) Close() error {
	return os.Remove(kh.Filename)
}
