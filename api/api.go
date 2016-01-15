package api

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Target struct {
	Hostname    string
	Fingerprint string
	Username    string
	DeployDir   string
	DeployCmd   string
}

type Config struct {
	Name         string `gorethink:"id"`
	NumServers   int
	InstanceType string
	PublicSSHKey string
}

func ValidateName(name string) error {
	// Make sure names are short enough to be stored in primary keys.
	if name == "" {
		return errors.New("Name empty")
	}
	return nil
}

func (c *Config) Validate() error {
	if err := ValidateName(c.Name); err != nil {
		return err
	}
	if c.NumServers == 0 {
		return errors.New("NumServers is 0")
	}
	// RSI: validate against list of legal instances.
	if c.InstanceType == "" {
		return fmt.Errorf("InstanceType `%s` is not legal.", c.InstanceType)
	}
	// RSI: consider checking that this is really a key.
	if c.PublicSSHKey == "" {
		return errors.New("PublicSSHKey not specified")
	}
	return nil
}

// RSI: documentation

type CreateResp struct {
	Target Target
}

type ReconfigResp struct {
	Target Target
}

type GetConfigReq struct {
	Name string
}

func (gc *GetConfigReq) Validate() error {
	return ValidateName(gc.Name)
}

type GetConfigResp struct {
	Config Config
	Target Target
}

type Resp struct {
	Success bool
	Error   string          `json:",omitempty"`
	Content json.RawMessage `json:",omitempty"`
}