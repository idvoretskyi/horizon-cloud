package db

import "github.com/rethinkdb/horizon-cloud/internal/api"

type Domain struct {
	Domain  string `gorethink:"id"`
	Project string
}

type Project struct {
	api.Project
	PublicKeys []string
}
