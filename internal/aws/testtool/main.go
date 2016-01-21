package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/fusion-ops/internal/aws"
	"github.com/rethinkdb/fusion-ops/internal/util"
)

var (
	start = flag.Bool("start", false, "start a new server")
	stop  = flag.Bool("stop", false, "stop all servers")
	ami   = flag.String("ami", "", "AMI to start")
)

func main() {
	flag.Parse()

	a := aws.New()

	if *start {
		log.Print("Starting new server")

		srv, err := a.StartServer("t2.micro", *ami)
		if err != nil {
			log.Fatal(err)
		}

		log.Print("Got new server ", srv)

		err = util.WaitConnectable("tcp", srv.PublicIP+":22", time.Minute*5)
		if err != nil {
			log.Fatal(err)
		}

		log.Print("Server is connectable!")
	}

	servers, err := a.ListServers()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Got", len(servers), "servers")
	for i, srv := range servers {
		fmt.Println("server ", i)
		spew.Dump(srv)
	}

	if *stop {
		for _, srv := range servers {
			if srv.State != "running" {
				continue
			}

			log.Printf("Stopping %v", srv.InstanceID)
			err := a.StopServer(srv.InstanceID)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	log.Print("done")
}
