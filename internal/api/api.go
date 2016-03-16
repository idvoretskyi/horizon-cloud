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

type User struct {
	Name          string `gorethink:"id"`
	PublicSSHKeys []string
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
	if err := validateProjectName(dc.Name, "Name"); err != nil {
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

type Domain struct {
	Domain  string `gorethink:"id"`
	Project string
}

func validateDomainName(domain string, fieldName string) error {
	// RSI: do more validation.
	if domain == "" {
		return fmt.Errorf("field `%s` empty", fieldName)
	}
	return nil
}

type Project struct {
	Name        string
	SSHAddress  string
	HTTPAddress string
}

func validateProjectName(name string, fieldName string) error {
	// RSI: more validation.
	if name == "" {
		return fmt.Errorf("field `%s` empty", fieldName)
	}
	return nil
}

func ProjectFromName(name string) Project {
	trueName := util.TrueName(name)
	return Project{
		Name:        name,
		SSHAddress:  "fs-" + trueName + ".user:22",
		HTTPAddress: "fn-" + trueName + ".user:80",
	}
}

// RSI: documentation

type ClusterStartBool bool

const AllowClusterStart ClusterStartBool = ClusterStartBool(true)
const DisallowClusterStart ClusterStartBool = ClusterStartBool(false)

////////////////////////////////////////////////////////////////////////////////
// EnsureConfigConnectable

var EnsureConfigConnectablePath = "/v1/configs/ensureConnectable"

type EnsureConfigConnectableReq struct {
	Name              string
	AllowClusterStart ClusterStartBool
}

func (r *EnsureConfigConnectableReq) Validate() error {
	err := validateProjectName(r.Name, "Name")
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

////////////////////////////////////////////////////////////////////////////////
// WaitConfigApplied

var WaitConfigAppliedPath = "/v1/configs/waitApplied"

type WaitConfigAppliedReq struct {
	Name    string
	Version string
}

func (wca *WaitConfigAppliedReq) Validate() error {
	if err := validateProjectName(wca.Name, "Name"); err != nil {
		return err
	}
	return nil
}

type WaitConfigAppliedResp struct {
	Config Config
	Target Target
}

////////////////////////////////////////////////////////////////////////////////
// SetConfig

var SetConfigPath = "/v1/configs/set"

type SetConfigReq struct {
	DesiredConfig
}

func (r *SetConfigReq) Validate() error {
	return r.DesiredConfig.Validate()
}

type SetConfigResp struct {
	Config
}

////////////////////////////////////////////////////////////////////////////////
// GetConfig

var GetConfigPath = "/v1/configs/get"

type GetConfigReq struct {
	Name string
}

func (r *GetConfigReq) Validate() error {
	return validateProjectName(r.Name, "Name")
}

type GetConfigResp struct {
	Config Config
}

////////////////////////////////////////////////////////////////////////////////
// UserCreate

var UserCreatePath = "/v1/users/create"

type UserCreateReq struct {
	Name string
}

func (r *UserCreateReq) Validate() error {
	return validateProjectName(r.Name, "Name")
}

type UserCreateResp struct{}

////////////////////////////////////////////////////////////////////////////////
// UserGet

var UserGetPath = "/v1/users/get"

type UserGetReq struct {
	Name string
}

func (r *UserGetReq) Validate() error {
	return validateProjectName(r.Name, "Name")
}

type UserGetResp struct {
	User User
}

////////////////////////////////////////////////////////////////////////////////
// UserAddKeys

var UserAddKeysPath = "/v1/users/addKeys"

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
	return validateProjectName(r.Name, "Name")
}

type UserAddKeysResp struct{}

////////////////////////////////////////////////////////////////////////////////
// UserDelKeys

var UserDelKeysPath = "/v1/users/delKeys"

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
	return validateProjectName(r.Name, "Name")
}

type UserDelKeysResp struct{}

////////////////////////////////////////////////////////////////////////////////
// SetDomain

var SetDomainPath = "/v1/domains/set"

type SetDomainReq struct {
	Domain
}

func (r *SetDomainReq) Validate() error {
	err := validateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}
	return validateDomainName(r.Domain.Domain, "Domain")
}

type SetDomainResp struct{}

////////////////////////////////////////////////////////////////////////////////
// GetDomainsByProject

var GetDomainsByProjectPath = "/v1/domains/getByProject"

type GetDomainsByProjectReq struct {
	Project string
}

func (r *GetDomainsByProjectReq) Validate() error {
	return validateProjectName(r.Project, "Project")
}

type GetDomainsByProjectResp struct {
	Domains []string
}

////////////////////////////////////////////////////////////////////////////////
// GetProjectsByKey

var GetProjectsByKeyPath = "/v1/projects/getByKey"

type GetProjectsByKeyReq struct {
	PublicKey string
}

func (gp *GetProjectsByKeyReq) Validate() error {
	if !ssh.ValidKey(gp.PublicKey) {
		return errors.New("invalid public key format")
	}
	return nil
}

type GetProjectsByKeyResp struct {
	Projects []Project
}

////////////////////////////////////////////////////////////////////////////////
// GetProjectByDomain

var GetProjectByDomainPath = "/v1/projects/getByDomain"

type GetProjectByDomainReq struct {
	Domain string
}

func (r *GetProjectByDomainReq) Validate() error {
	return validateDomainName(r.Domain, "Domain")
}

type GetProjectByDomainResp struct {
	Project *Project
}
