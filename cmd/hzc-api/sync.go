package main

import (
	"log"
	"sync"

	"golang.org/x/oauth2/jwt"

	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/gcloud"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/kube"
	"github.com/rethinkdb/horizon-cloud/internal/types"
)

var configs = make(map[string]*types.Config)
var configsLock sync.Mutex

func applyConfigs(serviceAccount *jwt.Config, rdb *db.DB, trueName string) {
	for {
		conf := func() *types.Config {
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

		// TODO: generalize
		gc, err := gcloud.New(serviceAccount, clusterName, "us-central1-f")
		if err != nil {
			// RSI: log serious
			log.Print(err)
			continue
		}

		k := kube.New(templatePath, gc)

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
		_, err = rdb.SetConfig(types.Config{
			ID:             conf.ID,
			AppliedVersion: conf.Version,
		})
		if err != nil {
			// RSI: log serious error.
			log.Printf("%s\n", err)
		}
	}
}

func configSync(ctx *hzhttp.Context) {
	rdb := ctx.DB()
	serviceAccount := ctx.ServiceAccount()

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
					go applyConfigs(serviceAccount, rdb, c.NewVal.ID)
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
