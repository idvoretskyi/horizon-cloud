package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/rethinkdb/fusion-ops/internal/api"
)

type config struct {
	APIClient *api.Client
	APISecret string
}

func main() {
	listenAddr := flag.String("listen", ":80", "Address to listen on")
	apiServer := flag.String("api-server", "http://localhost:8000", "API server base URL")
	apiServerSecret := flag.String("api-server-secret", "/secrets/api-shared-secret", "Path to API server shared secret")

	flag.Parse()

	conf := &config{}

	var err error

	conf.APIClient, err = api.NewClient(*apiServer)
	if err != nil {
		log.Fatalf("Couldn't create API client: %v", err)
	}

	secret, err := ioutil.ReadFile(*apiServerSecret)
	if err != nil {
		log.Fatalf("Couldn't read api server secret from %v: %v", *apiServerSecret, err)
	}
	conf.APISecret = string(secret)

	handler := NewHandler(conf)
	log.Fatal(http.ListenAndServe(*listenAddr, handler))
}
