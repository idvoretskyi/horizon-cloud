package main

import (
	"log"
	"time"

	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/aws"
	"github.com/rethinkdb/fusion-ops/internal/db"
	"github.com/rethinkdb/fusion-ops/internal/ssh"
	"github.com/rethinkdb/fusion-ops/internal/util"
)

var applyConfigCh = make(chan *applyConfigRequest, 8)

var fusionSeedAMI = "ami-50def93a"

func init() {
	go applyConfigWorker()
}

func applyConfigWorker() {
	for req := range applyConfigCh {
		<-req.Ready
		if !req.DoIt {
			continue
		}

		applyConfig(req.Config)
	}
}

func applyConfig(c api.Config) {
	cluster := aws.New(c.Name)

	servers, err := cluster.ListServers()
	if err != nil {
		// RSI: figure out how to tell the user this update will never happen
		log.Printf("Couldn't get instance list: %v", err)
	}

	if len(servers) == 0 {
		srv, err := cluster.StartServer(c.InstanceType, fusionSeedAMI)
		if err != nil {
			// RSI as above
			log.Printf("Couldn't start server: %v", err)
			return
		}

		err = util.WaitConnectable("tcp", srv.PublicIP+":22", time.Minute)
		if err != nil {
			log.Printf("Server %v never became accessible over ssh: %v",
				srv.InstanceID, err)
			return
		}

		servers = append(servers, srv)
	}

	errs := make(chan error)
	for i := range servers {
		go func(srv *aws.Server) {
			keys, err := ssh.KeyScan(srv.PublicIP)
			if err != nil {
				errs <- err
				return
			}

			kh, err := ssh.NewKnownHosts(keys)
			if err != nil {
				errs <- err
				return
			}
			defer kh.Close()

			client := ssh.New(ssh.Options{
				Host:       srv.PublicIP,
				User:       "ubuntu",
				KnownHosts: kh,
				// TODO: IdentityFile
			})

			err = client.RunCommand("sudo rm -f /home/*/.*history /root/.*history")
			if err != nil {
				errs <- err
				return
			}

			errs <- nil
		}(servers[i])
	}

	worked := true
	for range servers {
		err := <-errs
		if err != nil {
			log.Printf("Couldn't setup server: %v", err)
			worked = false
		}
	}

	if !worked {
		log.Printf("Didn't initialize, skipping config commit")
		return
	}

	err = rdb.SetConfig(&db.Config{
		Config:         c,
		AppliedVersion: c.Version,
	})
	if err != nil {
		log.Printf("Couldn't set config version in db after application: %v", err)
	}

	log.Printf("it worked!")
}
