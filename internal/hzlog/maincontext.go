package hzlog

import "os"

// MainLogger returns a new Logger with program-global log information filled
// in.
func MainLogger(progname string) (*Logger, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return BlankLogger().With(map[string]interface{}{
		"hostname": hostname,
		"program":  progname,
	}), nil
}
