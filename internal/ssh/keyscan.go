package ssh

import (
	"bytes"
	"fmt"
	"os/exec"
)

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
		out = append(out, string(line))
	}

	return out, nil
}
