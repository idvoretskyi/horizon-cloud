package main

import (
	"log"
	"sync"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/gcloud"
	"github.com/rethinkdb/horizon-cloud/internal/kube"
)

var configs = make(map[string]*api.Config)
var configsLock sync.Mutex

func applyConfigs(trueName string) {
	for {
		conf := func() *api.Config {
			configsLock.Lock()
			defer configsLock.Unlock()
			conf := configs[trueName]
			if conf == nil {
				delete(configs, trueName)
			} else {
				configs[trueName] = nil
			}
			return conf
		}()
		if conf == nil {
			break
		}

		gc, err := gcloud.New("horizon-cloud-1239", "us-central1-f") // TODO: generalize
		if err != nil {
			// RSI: log serious
			log.Print(err)
			continue
		}

		k := kube.New(gc)

		// RSI: tear down old project once we actually support changing
		// configurations.
		project, err := k.EnsureProject(*conf)

		if err != nil {
			// RSI: log serious error.
			log.Printf("%s\n", err)
			continue
		}
		log.Printf("waiting for config %s:%s", trueName, conf.Version)
		err = k.Wait(project)
		if err != nil {
			// RSI: log serious error
			log.Printf("%s\n", err)
			continue
		}
		log.Printf("successfully applied config %s:%s", trueName, conf.Version)
		err = rdb.SetConfig(api.Config{
			ID:             conf.ID,
			AppliedVersion: conf.Version,
		})
		if err != nil {
			// RSI: log serious error.
			log.Printf("%s\n", err)
		}
	}
}

func configSync(rdb *db.DB) {
	// RSI: shut down mid-spinup and see if it recovers.
	changeChan := make(chan db.ConfigChange)
	rdb.ConfigChanges(changeChan)
	for c := range changeChan {
		if c.NewVal != nil {
			if c.NewVal.AppliedVersion == c.NewVal.Version {
				continue
			}
			func() {
				configsLock.Lock()
				defer configsLock.Unlock()
				_, workerRunning := configs[c.NewVal.ID]
				configs[c.NewVal.ID] = c.NewVal
				if !workerRunning {
					go applyConfigs(c.NewVal.ID)
				}
			}()
		} else {
			if c.OldVal != nil {
				// RSI: tear down cluster.
			}
		}
	}
	panic("unreachable")
}
