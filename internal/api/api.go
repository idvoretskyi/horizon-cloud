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
	Name          string   `gorethink:"id,omitempty"`
	Version       string   `gorethink:",omitempty"`
	NumServers    int      `gorethink:",omitempty"`
	InstanceType  string   `gorethink:",omitempty"`
	PublicSSHKeys []string `gorethink:",omitempty"`
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
		return fmt.Errorf("InstanceType `%s` is not legal", c.InstanceType)
	}
	return nil
}

// RSI: documentation

type ClusterStartBool bool

const AllowClusterStart ClusterStartBool = ClusterStartBool(true)
const DisallowClusterStart ClusterStartBool = ClusterStartBool(false)

type EnsureConfigConnectableReq struct {
	Name              string
	Key               string
	AllowClusterStart ClusterStartBool
}

func (r *EnsureConfigConnectableReq) Validate() error {
	err := ValidateName(r.Name)
	if err != nil {
		return err
	}
	if r.Key == "" {
		return fmt.Errorf("Key empty")
	}
	// RSI: validate key
	return nil
}

type EnsureConfigConnectableResp struct {
	Config Config
}

type GetConfigReq struct {
	Name string
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

func writeJSON(rw http.ResponseWriter, code int, i interface{}) {
	// RSI: log errors from writeJSON.
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(i)
}

func WriteJSONError(rw http.ResponseWriter, code int, err error) {
	writeJSON(rw, code, resp{
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
	writeJSON(rw, code, r)
}

func ReadJSONResp(hr *http.Response, i interface{}) error {
	var resp resp
	err := json.NewDecoder(hr.Body).Decode(&resp)
	if err != nil {
		return err
	}
	if !resp.Success || hr.StatusCode != http.StatusOK {
		return fmt.Errorf("server error (%s): %s", hr.Status, resp.Error)
	}
	json.Unmarshal(*resp.Content, i)
	return nil
}
