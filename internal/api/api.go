package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/pborman/uuid"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

var ProjectEnvVarName = "HORIZON_PROJECT"

type Target struct {
	Hostname     string
	Fingerprints []string
	Username     string
	DeployDir    string
	DeployCmd    string
}

type DesiredConfig struct {
	Name         string `gorethink:",omitempty"`
	NumRDB       int    `gorethink:",omitempty"`
	SizeRDB      int    `gorethink:",omitempty"`
	NumHorizon   int    `gorethink:",omitempty"`
	NumFrontend  int    `gorethink:",omitempty"`
	SizeFrontend int    `gorethink:",omitempty"`
}

func (dc *DesiredConfig) Validate() error {
	if err := ValidateName(dc.Name); err != nil {
		return err
	}
	return nil
}

func DefaultDesiredConfig(name string) *DesiredConfig {
	return &DesiredConfig{
		Name:         name,
		NumRDB:       1,
		SizeRDB:      10,
		NumHorizon:   1,
		NumFrontend:  1,
		SizeFrontend: 1,
	}
}

type Config struct {
	DesiredConfig
	ID             string   `gorethink:"id,omitempty"`
	Version        string   `gorethink:",omitempty"`
	AppliedVersion string   `gorethink:",omitempty"`
	PublicSSHKeys  []string `gorethink:",omitempty"`
}

func ConfigFromDesired(dc *DesiredConfig) *Config {
	conf := Config{
		DesiredConfig: *dc,
		ID:            util.TrueName(dc.Name),
		Version:       uuid.New(),
	}
	return &conf
}

func ValidateName(name string) error {
	// RSI: more validation.
	if name == "" {
		return errors.New("Name empty")
	}
	return nil
}

// RSI: documentation

type ClusterStartBool bool

const AllowClusterStart ClusterStartBool = ClusterStartBool(true)
const DisallowClusterStart ClusterStartBool = ClusterStartBool(false)

type EnsureConfigConnectableReq struct {
	Name              string
	AllowClusterStart ClusterStartBool
}

func (r *EnsureConfigConnectableReq) Validate() error {
	err := ValidateName(r.Name)
	if err != nil {
		return err
	}
	return nil
}

type EnsureConfigConnectableResp struct {
	Config Config
}

type GetProjectsReq struct {
	SharedSecret string
	PublicKey    string
}

func (gp *GetProjectsReq) Validate() error {
	// RSI: validate key?
	return nil
}

type GetByAliasReq struct {
	SharedSecret string
	Alias        string
}

func (gp *GetByAliasReq) Validate() error {
	return nil
}

type Project struct {
	Name        string
	SSHAddress  string
	HTTPAddress string
}

func ProjectFromName(name string) Project {
	trueName := util.TrueName(name)
	return Project{
		Name:        name,
		SSHAddress:  "fs-" + trueName + ":22",
		HTTPAddress: "fn-" + trueName + ":80",
	}
}

type GetProjectsResp struct {
	Projects []Project
}

type GetByAliasResp struct {
	Project *Project
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
