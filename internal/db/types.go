package db

import "github.com/rethinkdb/fusion-ops/internal/api"

type Config struct {
	api.Config
	AppliedVersion string `gorethink:",omitempty"`
}