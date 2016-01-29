package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rethinkdb/fusion-ops/internal/aws"
	"github.com/rethinkdb/fusion-ops/internal/ssh"
	"github.com/rethinkdb/fusion-ops/internal/util"
)

func ensureEmpty(cluster *aws.AWS, allowStop bool) {
	servers, err := cluster.ListServers()
	if err != nil {
		log.Fatalf("Couldn't list servers: %v", err)
	}

	filtered := make([]*aws.Server, 0, len(servers))
	for _, srv := range servers {
		if srv.State == "running" {
			filtered = append(filtered, srv)
		}
	}
	servers = filtered

	if len(servers) > 0 {
		if !allowStop {
			log.Fatalf("A server is already running! Pass -allow-stop to terminate it")
		}

		for _, server := range servers {
			log.Printf("Terminating server %v", server.InstanceID)
			err := cluster.TerminateServer(server.InstanceID)
			if err != nil {
				log.Fatalf("Couldn't stop %s: %v", server.InstanceID, err)
			}
		}
	}
}

func startAMI(cluster *aws.AWS, baseAMI string) *aws.Server {
	log.Printf("Starting a new server")
	srv, err := cluster.StartServer("t2.medium", baseAMI)
	if err != nil {
		log.Fatalf("Couldn't start new server: %v", err)
	}

	log.Printf("Waiting for SSH access")
	err = util.WaitConnectable("tcp", srv.PublicIP+":22", time.Minute*5)
	if err != nil {
		log.Fatalf("Server never became accessible: %v", err)
	}

	return srv
}

func doSetup(srv *aws.Server, setupDir string) {
	log.Printf("Copying and running setup")

	keys, err := ssh.KeyScan(srv.PublicIP)
	if err != nil {
		log.Fatalf("Couldn't scan for ssh keys: %v", err)
	}

	kh, err := ssh.NewKnownHosts(keys)
	if err != nil {
		log.Fatalf("Couldn't create known_hosts file: %v", err)
	}
	defer kh.Close()

	client := ssh.New(ssh.Options{
		Host:       srv.PublicIP,
		User:       "ubuntu",
		KnownHosts: kh,
		// RSI: allow passing an IdentityFile
	})

	// rsync acts funny if you do anything but dir/ to dir/
	if !strings.HasSuffix(setupDir, "/") {
		setupDir = setupDir + "/"
	}

	err = client.RsyncTo(setupDir, "setup/", "")
	if err != nil {
		log.Fatalf("Couldn't rsync setup directory: %v", err)
	}

	err = client.RunCommand("sudo env RUNSETUP=YES setup/setup")
	if err != nil {
		log.Fatalf("Couldn't run setup script: %v", err)
	}

	log.Printf("Setup complete")
}

func imageAndTerminate(cluster *aws.AWS, srv *aws.Server, name string) string {
	log.Printf("Stopping server")
	err := cluster.StopServer(srv.InstanceID)
	if err != nil {
		log.Fatalf("Couldn't stop server: %v", err)
	}

	log.Printf("Creating image")
	ami, err := cluster.CreateImage(srv.InstanceID, name)
	if err != nil {
		log.Fatalf("Couldn't create image from instance: %v", err)
	}

	log.Printf("Terminating server")
	err = cluster.TerminateServer(srv.InstanceID)
	if err != nil {
		log.Fatalf("Couldn't terminate server: %v", err)
	}

	return ami
}

func checkDirExists(dir string) {
	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		log.Fatalf("Local directory `%s` does not exist", dir)
	}
	if err != nil {
		log.Fatalf("Couldn't stat `%s`: %v", dir, err)
	}
	if !fi.IsDir() {
		log.Fatalf("`%s` exists but is not a directory", dir)
	}
}

func main() {
	allowStop := flag.Bool("allow-stop", false, "Terminate running instances")
	baseAMI := flag.String("base-ami", "ami-fce3c696", "Base AMI to start with")
	setupDir := flag.String("setup-dir", "setup", "Path to the setup script directory")
	name := flag.String("name", "fusionseed", "The name of the new AMI")
	flag.Parse()

	checkDirExists(*setupDir)

	cluster := aws.New("fusion-seed")

	ensureEmpty(cluster, *allowStop)
	srv := startAMI(cluster, *baseAMI)
	doSetup(srv, *setupDir)
	ami := imageAndTerminate(cluster, srv, *name)

	log.Printf("SUCCESS! AMI created at %v with name %v", ami, *name)
}
