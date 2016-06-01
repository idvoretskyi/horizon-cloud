package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rethinkdb/horizon-cloud/internal/util"
)

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

type KubeConfig struct {
	NumRDB     int `gorethink:",omitempty"`
	SizeRDB    int `gorethink:",omitempty"`
	NumHorizon int `gorethink:",omitempty"`
}

func (dc *KubeConfig) Validate() error {
	if dc.NumRDB != 1 {
		return fmt.Errorf("NumRDB = %d, but only 1 is supported", dc.NumRDB)
	}
	if dc.SizeRDB < 10 {
		return fmt.Errorf("SizeRDB = %d, but only >=10 is supported", dc.SizeRDB)
	}
	if dc.NumHorizon != 1 {
		return fmt.Errorf("NumHorizon = %d, but only 1 is supported", dc.NumHorizon)
	}
	return nil
}

type HorizonConfig []byte

type Project struct {
	ID    string   `gorethink:"id,omitempty"`
	Name  string   `gorethink:",omitempty"`
	Users []string `gorethink:",omitempty"`

	Deleting bool `gorethink:",omitempty"`

	KubeConfig               KubeConfig `gorethink:",omitempty"`
	KubeConfigVersion        int64      `gorethink:",omitempty"`
	KubeConfigAppliedVersion int64      `gorethink:",omitempty"`

	HorizonConfig               HorizonConfig `gorethink:",omitempty"`
	HorizonConfigVersion        int64         `gorethink:",omitempty"`
	HorizonConfigAppliedVersion int64         `gorethink:",omitempty"`
	HorizonConfigLastError      string        `gorethink:",omitempty"`
	HorizonConfigErrorVersion   int64         `gorethink:",omitempty"`
}

type Domain struct {
	Domain  string `gorethink:"id"`
	Project string
}

type ProjectAddr struct {
	Name     string
	HTTPAddr string
}

func ProjectAddrFromName(name string) ProjectAddr {
	trueName := util.TrueName(name)
	return ProjectAddr{
		Name:     name,
		HTTPAddr: "h-" + trueName + ":8181",
	}
}

type ClusterStartBool bool

const AllowClusterStart ClusterStartBool = ClusterStartBool(true)
const DisallowClusterStart ClusterStartBool = ClusterStartBool(false)

type FileDescription struct {
	Path        string
	MD5         []byte
	ContentType string
}

func (d *FileDescription) Validate() error {
	if d.Path == "" {
		return errors.New("Path must not be empty")
	}
	if !util.IsSafeRelPath(d.Path) {
		return fmt.Errorf("Path %#v is not safe", d.Path)
	}
	if strings.HasPrefix(d.Path, ".well-known") {
		return fmt.Errorf("Path %#v is in .well-known, which is not supported")
	}
	if len(d.MD5) != 16 {
		return fmt.Errorf("MD5 must be exactly 16 bytes long (after base64 decoding)")
	}
	if d.ContentType == "" {
		return errors.New("ContentType must be set")
	}
	return nil
}

type FileUploadRequest struct {
	SourcePath string
	Method     string
	URL        string
	Headers    map[string]string
}
