package types

import (
	"fmt"

	"github.com/pborman/uuid"
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

type DesiredConfig struct {
	Name       string `gorethink:",omitempty"`
	NumRDB     int    `gorethink:",omitempty"`
	SizeRDB    int    `gorethink:",omitempty"`
	NumHorizon int    `gorethink:",omitempty"`

	// This is a pointer to slice because we need the zero value and
	// non-existence to be distinguishable.
	Users *[]string `gorethink:",omitempty"`
}

func (dc *DesiredConfig) Validate() error {
	if err := util.ValidateProjectName(dc.Name, "Name"); err != nil {
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

type Project struct {
	Name        string
	HTTPAddress string
}

func ProjectFromName(name string) Project {
	trueName := util.TrueName(name)
	return Project{
		Name:        name,
		HTTPAddress: "h-" + trueName + ".user:8181",
	}
}

type ClusterStartBool bool

const AllowClusterStart ClusterStartBool = ClusterStartBool(true)
const DisallowClusterStart ClusterStartBool = ClusterStartBool(false)
