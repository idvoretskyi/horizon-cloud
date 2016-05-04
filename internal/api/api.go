package api

import (
	"errors"
	"fmt"

	"github.com/rethinkdb/horizon-cloud/internal/ssh"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

var ProjectEnvVarName = "HORIZON_PROJECT"

// RSI: documentation

////////////////////////////////////////////////////////////////////////////////
// EnsureConfigConnectable

var EnsureConfigConnectablePath = "/v1/configs/ensureConnectable"

type EnsureConfigConnectableReq struct {
	Name              string
	AllowClusterStart types.ClusterStartBool
}

func (r *EnsureConfigConnectableReq) Validate() error {
	err := util.ValidateProjectName(r.Name, "Name")
	if err != nil {
		return err
	}
	if r.AllowClusterStart == types.AllowClusterStart {
		return fmt.Errorf("you are not authorized to start clusters")
	}
	return nil
}

type EnsureConfigConnectableResp struct {
	Config types.Config
}

////////////////////////////////////////////////////////////////////////////////
// SetConfig

var SetConfigPath = "/v1/configs/set"

type SetConfigReq struct {
	types.DesiredConfig
}

func (r *SetConfigReq) Validate() error {
	return r.DesiredConfig.Validate()
}

type SetConfigResp struct {
	types.Config
}

////////////////////////////////////////////////////////////////////////////////
// GetConfig

var GetConfigPath = "/v1/configs/get"

type GetConfigReq struct {
	Name string
}

func (r *GetConfigReq) Validate() error {
	return util.ValidateProjectName(r.Name, "Name")
}

type GetConfigResp struct {
	Config types.Config
}

////////////////////////////////////////////////////////////////////////////////
// UserCreate

var UserCreatePath = "/v1/users/create"

type UserCreateReq struct {
	Name string
}

func (r *UserCreateReq) Validate() error {
	return util.ValidateProjectName(r.Name, "Name")
}

type UserCreateResp struct{}

////////////////////////////////////////////////////////////////////////////////
// UserGet

var UserGetPath = "/v1/users/get"

type UserGetReq struct {
	Name string
}

func (r *UserGetReq) Validate() error {
	return util.ValidateProjectName(r.Name, "Name")
}

type UserGetResp struct {
	User types.User
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
	return util.ValidateProjectName(r.Name, "Name")
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
	return util.ValidateProjectName(r.Name, "Name")
}

type UserDelKeysResp struct{}

////////////////////////////////////////////////////////////////////////////////
// SetDomain

var SetDomainPath = "/v1/domains/set"

type SetDomainReq struct {
	types.Domain
}

func (r *SetDomainReq) Validate() error {
	err := util.ValidateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}
	return util.ValidateDomainName(r.Domain.Domain, "Domain")
}

type SetDomainResp struct{}

////////////////////////////////////////////////////////////////////////////////
// GetDomainsByProject

var GetDomainsByProjectPath = "/v1/domains/getByProject"

type GetDomainsByProjectReq struct {
	Project string
}

func (r *GetDomainsByProjectReq) Validate() error {
	return util.ValidateProjectName(r.Project, "Project")
}

type GetDomainsByProjectResp struct {
	Domains []string
}

////////////////////////////////////////////////////////////////////////////////
// GetUsersByKey

var GetUsersByKeyPath = "/v1/users/getByKey"

type GetUsersByKeyReq struct {
	PublicKey string
}

func (gp *GetUsersByKeyReq) Validate() error {
	if !ssh.ValidKey(gp.PublicKey) {
		return errors.New("invalid public key format")
	}
	return nil
}

type GetUsersByKeyResp struct {
	Users []string
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
	Projects []types.Project
}

////////////////////////////////////////////////////////////////////////////////
// GetProjectByDomain

var GetProjectByDomainPath = "/v1/projects/getByDomain"

type GetProjectByDomainReq struct {
	Domain string
}

func (r *GetProjectByDomainReq) Validate() error {
	return util.ValidateDomainName(r.Domain, "Domain")
}

type GetProjectByDomainResp struct {
	Project *types.Project
}

////////////////////////////////////////////////////////////////////////////////
// SetProjectHorizonConfig

var SetProjectHorizonConfigPath = "/v1/projects/setHorizonConfig"

type SetProjectHorizonConfigReq struct {
	Project string
	Config  string
}

func (r *SetProjectHorizonConfigReq) Validate() error {
	err := util.ValidateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}
	// RSI: Validate config at all?  Maybe a length limit?
	return nil
}

type SetProjectHorizonConfigResp struct{}

////////////////////////////////////////////////////////////////////////////////
// UpdateProjectManifest

var UpdateProjectManifestPath = "/v1/projects/updateManifest"

type UpdateProjectManifestReq struct {
	Token   string
	Project string
	Files   []types.FileDescription
}

func (r *UpdateProjectManifestReq) Validate() error {
	err := util.ValidateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}

	for _, file := range r.Files {
		err = file.Validate()
		if err != nil {
			return err
		}
	}

	if !util.ReasonableToken(r.Token) {
		return errors.New("Token is not of the correct form")
	}

	return nil
}

type UpdateProjectManifestResp struct {
	NeededRequests []types.FileUploadRequest
}
