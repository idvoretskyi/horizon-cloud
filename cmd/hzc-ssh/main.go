package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"

	"golang.org/x/crypto/ssh"
)

type config struct {
	HostKey   ssh.Signer
	ClientKey ssh.Signer
	APIClient *api.Client
}

func main() {
	listenAddr := flag.String("listen", ":10022", "Address to listen on")
	hostKeyPath := flag.String("host-key", "", "Path to private host key")
	clientKeyPath := flag.String("client-key", "", "Path to private client key")
	apiServer := flag.String("api-server", "http://localhost:8000", "API server base URL")
	apiServerSecret := flag.String("api-server-secret", "/secrets/api-shared-secret", "Path to API server shared secret")

	flag.Parse()

	log.SetFlags(log.Lshortfile)
	logger, err := hzlog.MainLogger("hzc-ssh")
	if err != nil {
		log.Fatal(err)
	}

	writerLogger := hzlog.WriterLogger(logger)
	defer writerLogger.Close()
	log.SetOutput(writerLogger)

	conf := &config{}

	secret, err := ioutil.ReadFile(*apiServerSecret)
	if err != nil {
		log.Fatalf("Couldn't read api server secret from %v: %v", *apiServerSecret, err)
	}

	conf.APIClient, err = api.NewClient(*apiServer, string(secret))
	if err != nil {
		log.Fatalf("Couldn't create API client: %v", err)
	}

	conf.HostKey, err = loadPrivateKey(*hostKeyPath)
	if err != nil {
		log.Fatalf("Couldn't read host key from %v: %v", *hostKeyPath, err)
	}

	conf.ClientKey, err = loadPrivateKey(*clientKeyPath)
	if err != nil {
		log.Fatalf("Couldn't read client key from %v: %v", *clientKeyPath, err)
	}

	l, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Couldn't listen on %v: %v", *listenAddr, err)
	}

	log.Printf("Now listening for new connections on %v", l.Addr())

	for {
		s, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Printf("Got temporary error %v on listening socket", err)
				time.Sleep(time.Second * 5)
				continue
			}
			log.Fatalf("Couldn't accept from socket: %v", err)
		}

		go handleClientConn(logger, s, conf)
		// TODO: consider connection count limits
	}
}
