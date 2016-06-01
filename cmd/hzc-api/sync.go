package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/gcloud"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/kube"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/spf13/viper"
)

var projects = make(map[string]*types.Project)
var projectsLock sync.Mutex

// Errors returned from this are shown to users.
func applyHorizonConfig(
	ctx *hzhttp.Context, k *kube.Kube, trueName string, hzc types.HorizonConfig) error {

	pods, err := k.GetHorizonPodsForProject(trueName)
	if err != nil {
		ctx.Error("%v", err)
		return fmt.Errorf("unable to get horizon pods for `%v`", trueName)
	}
	if len(pods) == 0 {
		err = fmt.Errorf("no pods found for `%v`", trueName)
		ctx.Error("%v", err)
		return err
	}
	pod := pods[rand.Intn(len(pods))]
	stdout, stderr, err := k.Exec(kube.ExecOptions{
		PodName: pod,
		In:      bytes.NewReader(hzc),
		Command: []string{"su", "-s", "/bin/sh", "horizon", "-c",
			"sleep 0.3; cat > /tmp/conf; echo stdout; echo stderr >&2"},
	})
	if err != nil {
		err = fmt.Errorf("Error setting Horizon config:\n"+
			"\nStdout:\n%v\n"+
			"\nStderr:\n%v\n"+
			"\nError:\n%v\n", stdout, stderr, err)
		return err
	}
	return nil
}

func applyProjects(ctx *hzhttp.Context, trueName string) {
	ctx = ctx.WithLog(map[string]interface{}{"action": "applyProjects"})
	for {
		conf := func() *types.Project {
			projectsLock.Lock()
			defer projectsLock.Unlock()
			conf := projects[trueName]
			if conf == nil {
				delete(projects, trueName)
			} else {
				projects[trueName] = nil
			}
			return conf
		}()
		if conf == nil {
			break
		}
		ctx := ctx.WithLog(map[string]interface{}{"project": conf.ID})
		ctx.Info("applying project")

		gc, err := gcloud.New(
			ctx.ServiceAccount(), viper.GetString("cluster_name"), "us-central1-f")
		if err != nil {
			ctx.Error("%v", err)
			continue
		}
		k := kube.New(viper.GetString("template_path"), viper.GetString("kube_namespace"), gc)

		if conf.Deleting {
			ctx.Info("deleting project")
			err := k.DeleteProject(conf.ID)
			ctx.MaybeError(err)
			err = ctx.DB().DeleteProject(conf.ID)
			ctx.MaybeError(err)
			continue
		}

		ctx.Info("KubeConfig: %v (applied: %v)",
			conf.KubeConfigVersion, conf.KubeConfigAppliedVersion)
		if conf.KubeConfigVersion != conf.KubeConfigAppliedVersion {
			project, err := k.EnsureProject(conf.ID, conf.KubeConfig)
			if err != nil {
				ctx.Error("%v", err)
				continue
			}
			log.Printf("waiting for Kube config %v:%v", trueName, conf.KubeConfigVersion)
			err = k.Wait(project)
			if err != nil {
				ctx.Error("%v", err)
				continue
			}
		}

		ctx.Info("HorizonConfig: %v (applied: %v)",
			conf.HorizonConfigVersion, conf.HorizonConfigAppliedVersion)
		if conf.HorizonConfigVersion != conf.HorizonConfigAppliedVersion {
			err = applyHorizonConfig(ctx, k, trueName, conf.HorizonConfig)
			if err != nil {
				ctx.Error("error applying Horizon config %v:%v (%v)",
					trueName, conf.HorizonConfigVersion, err)
				_, err = ctx.DB().UpdateProject(conf.ID, types.Project{
					HorizonConfigLastError:    err.Error(),
					HorizonConfigErrorVersion: conf.HorizonConfigVersion,
				})
				if err != nil {
					ctx.Error("%v", err)
				}
				continue
			}
		}

		ctx.Info("successfully applied project %v:%v/%v",
			trueName, conf.KubeConfigVersion, conf.HorizonConfigVersion)
		_, err = ctx.DB().UpdateProject(conf.ID, types.Project{
			KubeConfigAppliedVersion:    conf.KubeConfigVersion,
			HorizonConfigAppliedVersion: conf.HorizonConfigVersion,
		})
		if err != nil {
			ctx.Error("%v", err)
		}
	}
}

func projectSync(ctx *hzhttp.Context) {
	ctx = ctx.WithLog(map[string]interface{}{"action": "projectSync"})

	changeChan := make(chan db.ProjectChange)
	ctx.DB().ProjectChanges(changeChan)
	for c := range changeChan {
		if c.NewVal != nil {
			if c.NewVal.KubeConfigVersion == c.NewVal.KubeConfigAppliedVersion &&
				c.NewVal.HorizonConfigVersion == c.NewVal.HorizonConfigAppliedVersion &&
				!c.NewVal.Deleting {
				continue
			}
			func() {
				projectsLock.Lock()
				defer projectsLock.Unlock()
				_, workerRunning := projects[c.NewVal.ID]
				projects[c.NewVal.ID] = c.NewVal
				if !workerRunning {
					go applyProjects(ctx, c.NewVal.ID)
				}
			}()
		} else {
			if c.OldVal != nil {
				// TODO: tear down cluster.
			}
		}
	}
	panic("unreachable")
}
