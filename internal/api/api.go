package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Target struct {
	Hostname     string
	Fingerprints []string
	Username     string
	DeployDir    string
	DeployCmd    string
}

type Config struct {
	Name         string `gorethink:"id,omitempty"`
	Version      string `gorethink:",omitempty"`
	NumServers   int    `gorethink:",omitempty"`
	InstanceType string `gorethink:",omitempty"`
	PublicSSHKey string `gorethink:",omitempty"`
}

func ValidateName(name string) error {
	// Make sure names are short enough to be stored in primary keys.
	if name == "" {
		return errors.New("Name empty")
	}
	return nil
}

func ValidateVersion(version string) error {
	if version == "" {
		return errors.New("Version empty")
	}
	return nil
}

func (c *Config) Validate() error {
	if err := ValidateName(c.Name); err != nil {
		return err
	}
	if err := ValidateVersion(c.Version); err != nil {
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

type GetConfigReq struct {
	Name         string
	EnsureExists bool
}

func (gc *GetConfigReq) Validate() error {
	return ValidateName(gc.Name)
}

type GetConfigResp struct {
	Config Config
}

type WaitConfigAppliedReq struct {
	Name    string
	Version string
}

func (wca *WaitConfigAppliedReq) Validate() error {
	if err := ValidateName(wca.Name); err != nil {
		return err
	}
	if err := ValidateVersion(wca.Version); err != nil {
		return err
	}
	return nil
}

type WaitConfigAppliedResp struct {
	Config Config
	Target Target
}

type resp struct {
	Success bool
	Error   string           `json:",omitempty"`
	Content *json.RawMessage `json:",omitempty"`
}

func WriteJSON(rw http.ResponseWriter, code int, i interface{}) {
	// RSI: log errors from writeJSON.
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(i)
}

func WriteJSONError(rw http.ResponseWriter, code int, err error) {
	WriteJSON(rw, code, resp{
		Success: false,
		Error:   err.Error(),
	})
}

func WriteJSONResp(rw http.ResponseWriter, code int, i interface{}) {
	r := resp{Success: true}
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	rm := json.RawMessage(b)
	r.Content = &rm
	WriteJSON(rw, code, r)
}

func ReadJSONResp(hr *http.Response, i interface{}) error {
	var resp resp
	json.NewDecoder(hr.Body).Decode(&resp)
	if !resp.Success || hr.StatusCode != http.StatusOK {
		return fmt.Errorf("server error (%s): %s", hr.Status, resp.Error)
	}
	json.Unmarshal(*resp.Content, i)
	return nil
}
