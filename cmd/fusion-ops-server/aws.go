package main

import (
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/aws"
	"github.com/rethinkdb/fusion-ops/internal/db"
	"github.com/rethinkdb/fusion-ops/internal/ssh"
	"github.com/rethinkdb/fusion-ops/internal/util"
)

var fusionSeedAMI = "ami-b7f0d1dd"

func applyConfig(c api.Config, identityFile string) bool {
	// RSI: make this smart enough to only redeploy keys if that's all that changed.
	log.Printf("Applying config %s (version %s)...", c.Name, c.Version)
	defer log.Printf("...finished applying config %s (version %s).", c.Name, c.Version)
	cluster := aws.New(c.Name)

	servers, err := cluster.ListServers()
	if err != nil {
		// RSI: figure out how to tell the user this update will never happen
		log.Printf("Couldn't get instance list: %v", err)
		return false
	}

	filtered := make([]*aws.Server, 0, len(servers))
	for _, srv := range servers {
		if srv.State == "running" {
			filtered = append(filtered, srv)
		}
	}
	servers = filtered

	if len(servers) == 0 {
		srv, err := cluster.StartServer(c.InstanceType, fusionSeedAMI)
		if err != nil {
			// RSI as above
			log.Printf("Couldn't start server: %v", err)
			return false
		}

		err = util.WaitConnectable("tcp", srv.PublicIP+":22", time.Minute)
		if err != nil {
			log.Printf("Server %v never became accessible over ssh: %v",
				srv.InstanceID, err)
			return false
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
				Host:         srv.PublicIP,
				User:         "ubuntu",
				KnownHosts:   kh,
				IdentityFile: identityFile,
			})

			err = client.RsyncTo("instance-scripts/", "instance-scripts/")
			if err != nil {
				errs <- err
				return
			}

			cmd := client.Command("sudo ./instance-scripts/post-create")
			cmd.Stdin = strings.NewReader(strings.Join(c.PublicSSHKeys, "\n"))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
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
	return worked
}

func configToTarget(conf *db.Config) (*api.Target, error) {
	cluster := aws.New(conf.Name)

	servers, err := cluster.ListServers()
	if err != nil {
		return nil, err
	}

	var chosen *aws.Server
	for _, srv := range servers {
		if srv.State != "running" {
			continue
		}

		if chosen == nil || chosen.PublicIP < srv.PublicIP {
			chosen = srv
		}
	}
	if chosen == nil {
		return nil, errors.New("no applicable servers")
	}

	keys, err := ssh.KeyScan(chosen.PublicIP)
	if err != nil {
		return nil, errors.New("couldn't get public keys from server")
	}

	return &api.Target{
		Hostname:     chosen.PublicIP,
		Fingerprints: keys,
		Username:     "fusion",
		DeployDir:    "deploy/$version",
		DeployCmd:    "./post-deploy $version",
	}, nil
}
