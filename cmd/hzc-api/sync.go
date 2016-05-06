package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
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

// Errors returned from this are shown to users.
func applyHorizonConfig(k *kube.Kube, trueName string, hzc types.HorizonConfig) error {
	pods, err := k.GetHorizonPodsForProject(trueName)
	if err != nil {
		log.Print(err) // RSI: log serious error
		return fmt.Errorf("unable to get horizon pods for `%s`", trueName)
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found for `%s`", trueName) // RSI: log serious error
	}
	pod := pods[rand.Intn(len(pods))]
	stdout, stderr, err := k.Exec(kube.ExecOptions{
		PodName: pod,
		In:      bytes.NewReader([]byte(hzc)),
		Command: []string{"su", "-s", "/bin/sh", "horizon", "-c",
			"sleep 0.3; cat > /tmp/conf; echo stdout; echo stderr >&2"},
	})
	if err != nil {
		err = fmt.Errorf("Error setting Horizon config:\n"+
			"\nStdout:\n%s\n"+
			"\nStderr:\n%s\n"+
			"\nError:\n%v\n", stdout, stderr, err)
		return err
	}
	return nil
}

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

		gc, err := gcloud.New(serviceAccount, clusterName, "us-central1-f")
		if err != nil {
			log.Print(err) // RSI: log serious error
			continue
		}
		k := kube.New(templatePath, gc)

		if conf.KubeConfigVersion != conf.KubeConfigAppliedVersion {
			project, err := k.EnsureProject(conf.ID, conf.KubeConfig)
			if err != nil {
				log.Printf("%s\n", err) // RSI: log serious error.
				continue
			}
			log.Printf("waiting for Kube config %s:%s", trueName, conf.KubeConfigVersion)
			err = k.Wait(project)
			if err != nil {
				log.Print(err) // RSI: log serious error
				continue
			}
		}

		if conf.HorizonConfigVersion != conf.HorizonConfigAppliedVersion {
			err = applyHorizonConfig(k, trueName, conf.HorizonConfig)
			if err != nil {
				log.Printf("error applying Horizon config %s:%s (%v)",
					trueName, conf.HorizonConfigVersion, err)
				_, err = rdb.SetConfig(types.Config{
					ID: conf.ID,
					HorizonConfigLastError:    err.Error(),
					HorizonConfigErrorVersion: conf.HorizonConfigVersion,
				})
				if err != nil {
					log.Print(err) // RSI: log serious error
				}
				continue
			}
		}

		log.Printf("successfully applied config %s:%s/%s",
			trueName, conf.KubeConfigVersion, conf.HorizonConfigVersion)
		_, err = rdb.SetConfig(types.Config{
			ID: conf.ID,
			KubeConfigAppliedVersion:    conf.KubeConfigVersion,
			HorizonConfigAppliedVersion: conf.HorizonConfigVersion,
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
			if c.NewVal.KubeConfigVersion == c.NewVal.KubeConfigAppliedVersion &&
				c.NewVal.HorizonConfigVersion == c.NewVal.HorizonConfigAppliedVersion {
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
