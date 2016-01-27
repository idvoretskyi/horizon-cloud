package main

import (
	"log"

	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/db"
)

// RSI: shut down mid-spinup and see if it recovers.
func configSync(rdb *db.DB) {
	ch := make(chan *db.Config)
	rdb.ConfigChanges(ch)
	for conf := range ch {
		if conf.Version == conf.AppliedVersion {
			continue
		}
		// RSI: serialize this on a per-user basis instead of globally.
		worked := applyConfig(conf.Config)
		if !worked {
			// RSI: report stuff like this to an errors table that users can
			// read from.
			log.Printf("Unable to apply configuration.")
			continue
		}
		err := rdb.SetConfig(&db.Config{
			Config: api.Config{
				Name: conf.Name,
			},
			AppliedVersion: conf.Version,
		})
		if err != nil {
			log.Printf("SERIOUS ERROR: unable to update applied version (%s)", err)
		}
	}
	panic("unreachable")
}
