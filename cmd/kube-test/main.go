package main

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/rethinkdb/fusion-ops/internal/aws"
	"github.com/rethinkdb/fusion-ops/internal/kube"
)

func main() {
	log.SetFlags(log.Lshortfile)
	a := aws.New("ktest")
	vol, err := a.CreateVolume(32)
	if err != nil {
		log.Fatalf("%s", err)
	}
	log.Printf("created %v", vol)
	k := kube.New()
	rdb, err := k.CreateRDB("ktest2", vol.Id)
	if err != nil {
		err2 := a.DeleteVolume(vol.Id)
		if err2 != nil {
			log.Fatalf("%s (after %s)", err2, err)
		}
		log.Printf("deleted %v", vol)
		log.Fatalf("%s", err)
	}
	spew.Dump(rdb)
}
