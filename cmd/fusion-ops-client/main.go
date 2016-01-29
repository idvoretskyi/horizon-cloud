package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/ssh"
)

// RSI: we need a domain name.
var server = "http://localhost:8000"

type commandContext struct {
	Client            *api.Client
	ProjectName       string
	PublicSSHKey      string
	PrivateSSHKeyPath string
}

func buildCommandContext() *commandContext {
	client, err := api.NewClient(server)
	if err != nil {
		log.Fatal("Failed to initialize API client: %v", err)
	}

	var name string
	if len(os.Args) >= 3 {
		name = os.Args[2]
	} else {
		name = autoFindName()
	}

	// RSI: look for .fusion in an ancestor directory if it's not in cwd
	ensureDir(".fusion")
	key := ensureKey()

	return &commandContext{
		Client:            client,
		ProjectName:       name,
		PublicSSHKey:      key,
		PrivateSSHKeyPath: ".fusion/deploy_key", // TODO: don't hardcode this path
	}
}

func autoFindName() string {
	data, err := ioutil.ReadFile("package.json")
	if err != nil {
		log.Fatal("No project name specified and `package.json` does not exist.")
	}
	var npmPackage struct {
		Name string `json:"name"`
	}
	if err = json.Unmarshal(data, &npmPackage); err != nil {
		log.Fatalf("No project name specified and failed to parse `package.json`: %s", err)
	}
	if npmPackage.Name == "" {
		log.Fatal("No project name specified and `package.json` does not include one.")
	}
	return npmPackage.Name
}

func missing(f string) bool {
	_, err := os.Stat(f)
	miss := os.IsNotExist(err)
	if err != nil && !miss {
		log.Fatalf("Error statting `%s`: `%s`", f, err)
	}
	return miss
}

func ensureDir(s string) {
	err := os.Mkdir(s, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Unable to create directory `%s`.", s)
	}
}

func ensureKey() string {
	f1 := ".fusion/deploy_key"
	f2 := ".fusion/deploy_key.pub"
	privMissing := missing(f1)
	pubMissing := missing(f2)
	if privMissing != pubMissing {
		log.Fatalf("Only one of `%s` `%s` is present.", f1, f2)
	}
	if privMissing && pubMissing {
		cmd := exec.Command("ssh-keygen", "-q", "-f", f1, "-N", "")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Unable to create new key pair: %s", err)
		}
	}
	res, err := ioutil.ReadFile(f2)
	if err != nil {
		log.Fatalf("Unable to read key pair: %s", err)
	}
	return string(res)
}

func withSSHConnection(ctx *commandContext,
	fn func(*ssh.Client, *api.WaitConfigAppliedResp) error) error {

	// RSI: see if we can combine the two client API calls into one
	eccResp, err := ctx.Client.EnsureConfigConnectable(api.EnsureConfigConnectableReq{
		Name: ctx.ProjectName,
		Key:  ctx.PublicSSHKey,
	})
	if err != nil {
		log.Fatalf("failed to deploy: %s", err)
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
		KnownHosts:   kh,
		IdentityFile: ctx.PrivateSSHKeyPath,
	})

	return fn(sshClient, wcaResp)
}

func deployCommand(ctx *commandContext) {
	// RSI: sanity check dist (exists as dir, has index.html file in it)

	log.Printf("Deploying project...")
	err := withSSHConnection(ctx,
		func(sshClient *ssh.Client, wca *api.WaitConfigAppliedResp) error {
			log.Printf("Deploying to cluster:")
			spew.Dump(wca)

			vars := map[string]string{
				"version":   wca.Config.Version,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			lookupVar := func(s string) string { return vars[s] }

			err := sshClient.RsyncTo(
				"dist/",
				os.Expand(wca.Target.DeployDir, lookupVar),
				"../current/",
			)
			if err != nil {
				return fmt.Errorf("couldn't rsync to target: %v", err)
			}

			err = sshClient.RunCommand(os.Expand(wca.Target.DeployCmd, lookupVar))
			if err != nil {
				return fmt.Errorf("couldn't run post-deploy script: %v", err)
			}

			log.Printf("Deployed to http://%s/", wca.Target.Hostname)

			return nil
		})
	if err != nil {
		log.Fatalf("Failed to deploy: %v", err)
	}
}

func sshCommand(ctx *commandContext) {
	// RSI: this command should not start a new cluster
	err := withSSHConnection(ctx,
		func(sshClient *ssh.Client, wca *api.WaitConfigAppliedResp) error {
			return sshClient.RunInteractive()
		})
	if err != nil {
		log.Fatalf("SSH exited with failure: %v", err)
	}
}

func main() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) < 2 {
		log.Fatal("No subcommand specified.")
	}

	ctx := buildCommandContext()

	switch os.Args[1] {
	case "deploy":
		deployCommand(ctx)
	case "ssh":
		sshCommand(ctx)

	default:
		log.Fatalf("Unrecognized subcommand `%s`.", os.Args[1])
	}
}
