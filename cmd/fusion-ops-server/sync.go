package main

import (
	"log"
	"sync"

	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/db"
	"github.com/rethinkdb/fusion-ops/internal/kube"
)

var configs = make(map[string]*api.Config)
var configsLock sync.Mutex

func applyConfigs(name string) {
	for {
		conf := func() *api.Config {
			configsLock.Lock()
			defer configsLock.Unlock()
			conf := configs[name]
			if conf == nil {
				delete(configs, name)
			} else {
				configs[name] = nil
			}
			return conf
		}()
		if conf == nil {
			break
		}
		// RSI: what should the cluster name be exactly?  I don't quite
		// understand the semantics here.
		k := kube.New("horizon")
		// RSI: tear down old project once we actually support changing
		// configurations.
		project, err := k.CreateProject(*conf)
		if err != nil {
			// RSI: log serious error.
			log.Printf("%s\n", err)
			continue
		}
		log.Printf("waiting for config %s:%s", name, conf.Version)
		err = k.Wait(project)
		if err != nil {
			// RSI: log serious error
			log.Printf("%s\n", err)
			continue
		}
		log.Printf("successfully applied config %s:%s", name, conf.Version)
		err = rdb.SetConfig(&db.Config{
			Config: api.Config{
				Name: conf.Name,
			},
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
				_, workerRunning := configs[c.NewVal.Name]
				configs[c.NewVal.Name] = &c.NewVal.Config
				if !workerRunning {
					go applyConfigs(c.NewVal.Name)
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
