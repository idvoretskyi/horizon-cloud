package cmd

import (
	"log"
	"os"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/ssh"
)

type commandContext struct {
	Client            *api.Client
	ProjectName       string
	PrivateSSHKeyPath string
}

func ensureDir(s string) {
	err := os.Mkdir(s, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Unable to create directory `%s`.", s)
	}
}

func missing(f string) bool {
	_, err := os.Stat(f)
	miss := os.IsNotExist(err)
	if err != nil && !miss {
		log.Fatalf("Error statting `%s`: `%s`", f, err)
	}
	return miss
}

func withSSHConnection(
	ctx *commandContext,
	AllowClusterStart api.ClusterStartBool,
	fn func(*ssh.Client, *api.WaitConfigAppliedResp) error) error {

	// RSI: see if we can combine the two client API calls into one
	eccResp, err := ctx.Client.EnsureConfigConnectable(api.EnsureConfigConnectableReq{
		Name:              ctx.ProjectName,
		AllowClusterStart: AllowClusterStart,
	})
	if err != nil {
		log.Fatalf("failed: %s", err)
	}

	log.Printf("Waiting for cluster to become ready...")
	wcaResp, err := ctx.Client.WaitConfigApplied(api.WaitConfigAppliedReq{
		Name:    ctx.ProjectName,
		Version: eccResp.Config.Version,
	})
	if err != nil {
		log.Fatalf("Failed to wait for cluster: %v", err)
	}

	kh, err := ssh.NewKnownHosts(wcaResp.Target.Fingerprints)
	if err != nil {
		log.Fatalf("Failed to create known_hosts file: %v", err)
	}
	defer kh.Close()

	sshClient := ssh.New(ssh.Options{
		Host:         wcaResp.Target.Hostname,
		User:         wcaResp.Target.Username,
		Environment:  map[string]string{api.ProjectEnvVarName: ctx.ProjectName},
		KnownHosts:   kh,
		IdentityFile: ctx.PrivateSSHKeyPath,
	})

	return fn(sshClient, wcaResp)
}
