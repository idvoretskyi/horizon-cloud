package api

import (
	"errors"

	"github.com/rethinkdb/horizon-cloud/internal/ssh"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

var ProjectEnvVarName = "HORIZON_PROJECT"

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
	ProjectID     types.ProjectID
	Files         []types.FileDescription
	HorizonConfig types.HorizonConfig
}

func (r *UpdateProjectManifestReq) Validate() error {
	err := r.ProjectID.Validate()
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

////////////////////////////////////////////////////////////////////////////////
// GetProjectsByToken

var GetProjectsByTokenPath = "/v1/projects/getByToken"

type GetProjectsByTokenReq struct {
	Token string
}

func (r *GetProjectsByTokenReq) Validate() error {
	if !util.ReasonableToken(r.Token) {
		return errors.New("Token is not of the correct form")
	}
	return nil
}

type GetProjectsByTokenResp struct {
	Projects []*types.Project
}
