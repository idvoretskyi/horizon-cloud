package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"

	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/kube"
	"github.com/rethinkdb/horizon-cloud/internal/types"
)

var projects = make(map[string]*types.Project)
var projectsLock sync.Mutex

func applyHorizonConfig(
	// Errors returned from this are shown to users.
	k *kube.Kube, ctx *hzhttp.Context, conf *types.Project) error {
	ctx.Info("Applying Horizon config: %#v", conf.HorizonConfig)

	hzc := conf.HorizonConfig
	pods, err := k.GetHorizonPodsForProject(conf.ID)
	if err != nil {
		ctx.Error("%v", err)
		return fmt.Errorf("unable to get horizon pods")
	}
	if len(pods) == 0 {
		err = fmt.Errorf("no pods found")
		ctx.Error("%v", err)
		return err
	}
	pod := pods[rand.Intn(len(pods))]
	stdout, stderr, err := k.Exec(kube.ExecOptions{
		PodName: pod,
		In:      bytes.NewReader(hzc),
		Command: []string{"su", "-s", "/bin/sh", "horizon", "-c",
			"hz set-schema -n app -"},
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

func applyKubeConfig(
	// Errors returned from this are shown to users.
	k *kube.Kube, ctx *hzhttp.Context, conf *types.Project) error {
	ctx.Info("Applying Kube config: %#v", conf.KubeConfig)
	project, err := k.EnsureProject(conf.ID, conf.KubeConfig)
	if err != nil {
		ctx.Error(err.Error())
		return fmt.Errorf("error applying Kube config")
	}
	ctx.Info("waiting for Kube config")
	err = k.Wait(project)
	if err != nil {
		ctx.Error(err.Error())
		return fmt.Errorf("error waiting for Kube config")
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
		k := ctx.Kube
		if conf.Deleting {
			ctx.Info("deleting project")
			err := k.DeleteProject(conf.ID)
			ctx.MaybeError(err)
			err = ctx.DB().DeleteProject(conf.ID)
			ctx.MaybeError(err)
			continue
		}
		ctx.Info("KubeConfigVersion: %#v", conf.KubeConfigVersion)
		kConfVer := conf.KubeConfigVersion.MaybeConfigure(func() error {
			return applyKubeConfig(k, ctx, conf)
		})
		ctx.Info("new KubeConfigVersion: %#v", kConfVer)
		ctx.Info("HorizonConfigVersion: %#v", conf.HorizonConfigVersion)
		hzConfVer := conf.HorizonConfigVersion.MaybeConfigure(func() error {
			return applyHorizonConfig(k, ctx, conf)
		})
		ctx.Info("new HorizonConfigVersion: %#v", hzConfVer)
		_, err := ctx.DB().UpdateProject(conf.ID, types.Project{
			KubeConfigVersion:    kConfVer,
			HorizonConfigVersion: hzConfVer,
		})
		ctx.MaybeError(err)
		ctx.Info("done applying project")
	}
}

func projectSync(ctx *hzhttp.Context) {
	ctx = ctx.WithLog(map[string]interface{}{"action": "projectSync"})

	changeChan := make(chan db.ProjectChange)
	ctx.DB().ProjectChanges(changeChan)
	for c := range changeChan {
		if c.NewVal != nil {
			if c.NewVal.KubeConfigVersion.Desired == c.NewVal.KubeConfigVersion.Applied &&
				c.NewVal.HorizonConfigVersion.Desired == c.NewVal.HorizonConfigVersion.Applied &&
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
