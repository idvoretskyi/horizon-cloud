package db

import "github.com/rethinkdb/horizon-cloud/internal/api"

type Alias struct {
	Alias   string `gorethink:"id"`
	Project string
}

type Project struct {
	api.Project
	PublicKeys []string
}
