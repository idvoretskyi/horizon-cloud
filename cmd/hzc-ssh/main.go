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
	HostKey     ssh.Signer
	APIClient   *api.Client
	TokenSecret []byte
}

func main() {
	listenAddr := flag.String("listen", ":10022", "Address to listen on")
	hostKeyPath := flag.String("host-key", "/secrets/ssh-proxy-keys/host-rsa", "Path to private host key")
	apiServer := flag.String("api-server", "http://localhost:8000", "API server base URL")
	apiServerSecret := flag.String("api-server-secret", "/secrets/api-shared-secret/api-shared-secret", "Path to API server shared secret")
	tokenSecretPath := flag.String("token-secret", "/secrets/token-secret/token-secret", "Path to token shared secret")

	flag.Parse()

	log.SetFlags(log.Lshortfile)
	logger, err := hzlog.MainLogger("hzc-ssh")
	if err != nil {
		log.Fatal(err)
	}

	writerLogger := hzlog.WriterLogger(logger)
	log.SetOutput(writerLogger)

	conf := &config{}

	apiSecret, err := ioutil.ReadFile(*apiServerSecret)
	if err != nil {
		log.Fatalf("Couldn't read api server secret from %v: %v", *apiServerSecret, err)
	}

	tokenSecret, err := ioutil.ReadFile(*tokenSecretPath)
	if err != nil {
		log.Fatalf("Couldn't read token secret from %v: %v", *tokenSecretPath, err)
	}
	conf.TokenSecret = tokenSecret

	conf.APIClient, err = api.NewClient(*apiServer, string(apiSecret))
	if err != nil {
		log.Fatalf("Couldn't create API client: %v", err)
	}

	conf.HostKey, err = loadPrivateKey(*hostKeyPath)
	if err != nil {
		log.Fatalf("Couldn't read host key from %v: %v", *hostKeyPath, err)
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
