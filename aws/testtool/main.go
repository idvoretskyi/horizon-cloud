package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/fusion-ops/aws"
)

var (
	start = flag.Bool("start", false, "start a new server")
	stop  = flag.Bool("stop", false, "stop all servers")
)

func main() {
	flag.Parse()

	a := aws.New()

	if *start {
		log.Print("Starting new server")
		srv, err := a.StartServer("t1.micro")
		if err != nil {
			log.Fatal(err)
		}

		log.Print("Got new server ", srv)
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
			log.Printf("Stopping %v", srv.InstanceID)
			err := a.StopServer(srv.InstanceID)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	log.Print("done")
}
