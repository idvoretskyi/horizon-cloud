package main

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/fusion-ops/internal/kube"
)

func main() {
	log.SetFlags(log.Lshortfile)
	k := kube.New("cluster")
	proj, err := k.CreateProject("ktest")
	if err != nil {
		log.Printf("%s", err)
		log.Printf("Hanging forever to let goroutines finish...")
		ch := make(chan error)
		spew.Dump(<-ch)
	}
	spew.Dump(proj)
}
