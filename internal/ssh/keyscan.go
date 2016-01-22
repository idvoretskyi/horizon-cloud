package ssh

import (
	"bytes"
	"fmt"
	"os/exec"
)

// KeyScan scans the given host for its host keys and returns them as a list of
// strings, each one of which represents a line from a known_hosts file.
func KeyScan(host string) ([]string, error) {
	var buf bytes.Buffer

	cmd := exec.Command("ssh-keyscan", host)
	cmd.Stdout = &buf

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("Couldn't scan for keys on host `%v`", host)
	}

	var out []string
	for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			out = append(out, string(line))
		}
	}

	return out, nil
}
