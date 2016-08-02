package api

import (
	"errors"

	"github.com/rethinkdb/horizon-cloud/internal/ssh"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

var ProjectEnvVarName = "HORIZON_PROJECT"

////////////////////////////////////////////////////////////////////////////////
// SetDomain

var SetDomainPath = "/v1/domains/set"

type SetDomainReq struct {
	types.Domain
}

// RSI: rip this out.
func (r *SetDomainReq) Validate() error {
	return util.ValidateDomainName(r.Domain.Domain, "Domain")
}

type SetDomainResp struct{}

////////////////////////////////////////////////////////////////////////////////
// DelDomain

var DelDomainPath = "/v1/domains/del"

type DelDomainReq struct {
	types.Domain
}

// RSI: rip this out.
func (r *DelDomainReq) Validate() error {
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
