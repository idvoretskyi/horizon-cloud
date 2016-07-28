package api

import (
	"errors"
	"fmt"

	"github.com/rethinkdb/horizon-cloud/internal/ssh"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

var ProjectEnvVarName = "HORIZON_PROJECT"

////////////////////////////////////////////////////////////////////////////////
// SetProjectKubeConfig

var SetProjectKubeConfigPath = "/v1/projects/setKubeConfig"

type SetProjectKubeConfigReq struct {
	Project    string
	KubeConfig types.KubeConfig
}

func (r *SetProjectKubeConfigReq) Validate() error {
	err := util.ValidateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}
	return r.KubeConfig.Validate()
}

type SetProjectKubeConfigResp struct {
	types.Project
}

////////////////////////////////////////////////////////////////////////////////
// DeleteProject

var DeleteProjectPath = "/v1/projects/delete"

type DeleteProjectReq struct {
	Project string
}

func (r *DeleteProjectReq) Validate() error {
	return util.ValidateProjectName(r.Project, "Project")
}

type DeleteProjectResp struct{}

////////////////////////////////////////////////////////////////////////////////
// AddProjectUsers

var AddProjectUsersPath = "/v1/projects/addUsers"

type AddProjectUsersReq struct {
	Project string
	Users   []string
}

func (r *AddProjectUsersReq) Validate() error {
	err := util.ValidateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}
	if len(r.Users) == 0 {
		return fmt.Errorf("no users specified")
	}
	for _, user := range r.Users {
		err := util.ValidateUserName(user)
		if err != nil {
			return err
		}
	}
	return nil
}

type AddProjectUsersResp struct {
	types.Project
}

////////////////////////////////////////////////////////////////////////////////
// DelProjectUsers

var DelProjectUsersPath = "/v1/projects/delUsers"

type DelProjectUsersReq struct {
	Project string
	Users   []string
}

func (r *DelProjectUsersReq) Validate() error {
	err := util.ValidateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}
	if len(r.Users) == 0 {
		return fmt.Errorf("no users specified")
	}
	for _, user := range r.Users {
		err := util.ValidateUserName(user)
		if err != nil {
			return err
		}
	}
	return nil
}

type DelProjectUsersResp struct {
	types.Project
}

////////////////////////////////////////////////////////////////////////////////
// GetProject

var GetProjectPath = "/v1/projects/get"

type GetProjectReq struct {
	Project string
}

func (r *GetProjectReq) Validate() error {
	return util.ValidateProjectName(r.Project, "Project")
}

type GetProjectResp struct {
	types.Project
}

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
// DelDomain

var DelDomainPath = "/v1/domains/del"

type DelDomainReq struct {
	types.Domain
}

func (r *DelDomainReq) Validate() error {
	err := util.ValidateProjectName(r.Project, "Project")
	if err != nil {
		return err
	}
	return util.ValidateDomainName(r.Domain.Domain, "Domain")
}

type DelDomainResp struct{}

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
// GetProjectAddrsByKey

var GetProjectAddrsByKeyPath = "/v1/projects/getAddrsByKey"

type GetProjectAddrsByKeyReq struct {
	PublicKey string
}

func (gp *GetProjectAddrsByKeyReq) Validate() error {
	if !ssh.ValidKey(gp.PublicKey) {
		return errors.New("invalid public key format")
	}
	return nil
}

type GetProjectAddrsByKeyResp struct {
	ProjectAddrs []types.ProjectAddr
}

////////////////////////////////////////////////////////////////////////////////
// GetProjectAddrByDomain

var GetProjectAddrByDomainPath = "/v1/projects/getByDomain"

type GetProjectAddrByDomainReq struct {
	Domain string
}

func (r *GetProjectAddrByDomainReq) Validate() error {
	return util.ValidateDomainName(r.Domain, "Domain")
}

type GetProjectAddrByDomainResp struct {
	ProjectAddr *types.ProjectAddr
}

////////////////////////////////////////////////////////////////////////////////
// UpdateProjectManifest

var UpdateProjectManifestPath = "/v1/projects/updateManifest"

type UpdateProjectManifestReq struct {
	Token         string
	Project       string
	Files         []types.FileDescription
	HorizonConfig types.HorizonConfig
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

	// TODO: validate HorizonConfig

	return nil
}

type UpdateProjectManifestResp struct {
	NeededRequests []types.FileUploadRequest
}
