package db

import "github.com/rethinkdb/horizon-cloud/internal/api"

type Project struct {
	api.Project
	PublicKeys []string
}
