package main

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/fusion-ops/internal/kube"
)

func main() {
	log.SetFlags(log.Lshortfile)
	k := kube.New("cluster")
	proj, err := k.CreateProject("ktest3")
	if err != nil {
		log.Fatalf("%s", err)
	}
	spew.Dump(proj)
}
