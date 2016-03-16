package api

import (
	"errors"
	"fmt"

	"github.com/pborman/uuid"
	"github.com/rethinkdb/horizon-cloud/internal/ssh"
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

	// This is a pointer to slice because we need the zero value and
	// non-existence to be distinguishable.
	Users *[]string `gorethink:",omitempty"`
}

func (dc *DesiredConfig) Validate() error {
	if err := ValidateName(dc.Name); err != nil {
		return err
	}
	if dc.NumRDB != 1 {
		return fmt.Errorf("NumRDB = %d, but only 1 is supported", dc.NumRDB)
	}
	if dc.SizeRDB < 10 {
		return fmt.Errorf("SizeRDB = %d, but only >=10 is supported", dc.SizeRDB)
	}
	if dc.NumHorizon != 1 {
		return fmt.Errorf("NumHorizon = %d, but only 1 is supported", dc.NumHorizon)
	}
	if dc.NumFrontend != 1 {
		return fmt.Errorf("NumFrontend = %d, but only 1 is supported", dc.NumFrontend)
	}
	if dc.SizeFrontend < 10 {
		return fmt.Errorf("SizeFrontend = %d, but only >=10 is supported", dc.SizeFrontend)
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
	ID             string `gorethink:"id,omitempty"`
	Version        string `gorethink:",omitempty"`
	AppliedVersion string `gorethink:",omitempty"`
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
	if r.AllowClusterStart == AllowClusterStart {
		return fmt.Errorf("you are not authorized to start clusters")
	}
	return nil
}

type EnsureConfigConnectableResp struct {
	Config Config
}

type GetProjectsReq struct {
	PublicKey string
}

func (gp *GetProjectsReq) Validate() error {
	if !ssh.ValidKey(gp.PublicKey) {
		return errors.New("invalid public key format")
	}
	return nil
}

type GetByAliasReq struct {
	Alias string
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
		SSHAddress:  "fs-" + trueName + ".user:22",
		HTTPAddress: "fn-" + trueName + ".user:80",
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

type UserCreateReq struct {
	Name string
}

func (r *UserCreateReq) Validate() error {
	return ValidateName(r.Name)
}

type UserCreateResp struct {
}

type UserGetReq struct {
	Name string
}

func (r *UserGetReq) Validate() error {
	return ValidateName(r.Name)
}

type User struct {
	Name          string `gorethink:"id"`
	PublicSSHKeys []string
}

type UserGetResp struct {
	User User
}

type UserAddKeysReq struct {
	Name string
	Keys []string
}

func (r *UserAddKeysReq) Validate() error {
	for _, key := range r.Keys {
		if !ssh.ValidKey(key) {
			return errors.New("invalid public key format")
		}
	}
	return ValidateName(r.Name)
}

type UserAddKeysResp struct {
}

type UserDelKeysReq struct {
	Name string
	Keys []string
}

func (r *UserDelKeysReq) Validate() error {
	for _, key := range r.Keys {
		if !ssh.ValidKey(key) {
			return errors.New("invalid public key format")
		}
	}
	return ValidateName(r.Name)
}

type UserDelKeysResp struct {
}
