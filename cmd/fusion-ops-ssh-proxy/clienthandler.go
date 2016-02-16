package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rethinkdb/fusion-ops/internal/api"

	"golang.org/x/crypto/ssh"
)

var (
	projectEnvVarName = []byte("FUSION_PROJECT")
)

func handleClient(s net.Conn, c *config) {
	logPrefix := strings.Replace(
		fmt.Sprintf("[%v <-> %v] ", s.RemoteAddr(), s.LocalAddr()),
		"%", "%%", -1)
	logger := func(f string, i ...interface{}) { log.Printf(logPrefix+f, i...) }

	logger("Accepted new connection")
	defer logger("Done with connection")
	defer s.Close()

	projectTargets := []*api.Project{
		{"fusiondev", "127.0.0.1:22"},
	}

	serverConfig := &ssh.ServerConfig{
		ServerVersion: "SSH-2.0-FusionOpsProxy",
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() != "fusion" {
				return nil, errors.New("Username must be 'fusion'")
			}
			return nil, nil
		},
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			if err == nil {
				logger("Authentication method=%v succeeded", method)
			} else {
				logger("Authentication method=%v failed: %v", method, err)
			}
		},
	}

	serverConfig.AddHostKey(c.HostKey)

	serverConn, chans, reqs, err := ssh.NewServerConn(s, serverConfig)
	if err != nil {
		logger("Failed to set up ssh connection: %v", err)
		return
	}

	logger("Handshake complete, ClientVersion=%#v",
		string(serverConn.ClientVersion()))

	go ssh.DiscardRequests(reqs)

	var wg sync.WaitGroup
	defer wg.Wait()

	for newCh := range chans {
		upstreamType := newCh.ChannelType()
		upstreamExtra := newCh.ExtraData()

		if upstreamType != "session" {
			logger("Rejecting channel of type %v", newCh.ChannelType())
			newCh.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newCh.Accept()
		if err != nil {
			logger("Error accepting new channel: %v", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer channel.Close()

			var projectName string

			envTimeout := time.NewTimer(time.Second * 15)
			defer envTimeout.Stop()

			var pendingRequests []*ssh.Request
			defer func() {
				for _, req := range pendingRequests {
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}()

			// Phase 1: Read requests to look for an env variable referencing
			// which project to route to.

		ENVREAD:
			for {
				select {
				case <-envTimeout.C:
					// Took too long to get to a terminal request. Give up.
					fmt.Fprintf(channel.Stderr(),
						"Took too long to get project environment variable\n")
					return
				case req, ok := <-requests:
					if !ok {
						// requests channel closed, remote side probably closed
						// the channel. In any case, nothing to do.
						return
					}

					logger("request type=%v, extra=%#v", req.Type, string(req.Payload))

					pendingRequests = append(pendingRequests, req)
					switch req.Type {
					case "env":
						key, value, err := decodeLengthPrefixedKV(req.Payload)
						if err != nil {
							continue
						}
						if bytes.Equal(key, projectEnvVarName) {
							projectName = string(value)
							break ENVREAD
						}

					case "shell", "exec":
						// Final requests; we've definitely seen all the
						// environment variables that are wanted, so we can move
						// to the next phase of proxying.
						break ENVREAD
					}
				}
			}
			envTimeout.Stop()

			logger("got projectName = %#v", projectName)

			// Phase 2: Verify that the project name was given and is valid.

			var projectErr error
			var project *api.Project
			if projectName == "" {
				projectErr = errors.New("No project name passed")
			} else {
				for _, proj := range projectTargets {
					if proj.Name == projectName {
						project = proj
						break
					}
				}
				if project == nil {
					projectErr = fmt.Errorf("Project `%v` not found", projectName)
				}
			}

			if projectErr != nil {
				fmt.Fprintf(channel.Stderr(),
					"Couldn't find an appropriate target: %v\n", projectErr)
				return
			}

			// Phase 3: Connect to target.

			logger("Connecting to %v (project name %#v)",
				project.Address, project.Name)

			clientNet, err := net.Dial("tcp", project.Address)
			if err != nil {
				logger("Couldn't connect to %v: %v", project.Address, err)
				fmt.Fprintf(channel.Stderr(),
					"Couldn't connect to server hosting project `%v`\n", project.Name)
				return
			}
			defer clientNet.Close()

			clientConfig := &ssh.ClientConfig{
				User: "ckastorff",
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(c.ClientKey),
				},
				ClientVersion: "SSH-2.0-FusionOpsProxy",
			}

			clientConn, clientChans, clientReqs, err :=
				ssh.NewClientConn(clientNet, project.Address, clientConfig)
			if err != nil {
				logger("Couldn't setup SSH connection to %v: %v", project.Address, err)
				fmt.Fprintf(channel.Stderr(),
					"Couldn't connect to server hosting project `%v`\n", project.Name)
				return
			}

			go ssh.DiscardRequests(clientReqs)

			go func() {
				for newCh := range clientChans {
					logger("Rejecting upstream channel request")
					newCh.Reject(ssh.Prohibited, "prohibited")
				}
			}()

			logger("Connected to upstream, creating client channel.")

			clientChannel, clientChannelReqs, err := clientConn.OpenChannel(
				upstreamType, upstreamExtra)
			if err != nil {
				logger("Couldn't create client channel: %v", err)
				fmt.Fprintf(channel.Stderr(),
					"Couldn't connect to server hosting project `%v`\n", project.Name)
				return
			}

			// Phase 4: Forward channel requests and data.

			logger("Client channel ready, forwarding requests and data.")

			for _, req := range pendingRequests {
				ok, err := clientChannel.SendRequest(req.Type, req.WantReply, req.Payload)
				if err != nil {
					logger("Error while forwarding pending request: %v", err)
					return
				}

				if req.WantReply {
					req.Reply(ok, nil)
				}
			}
			pendingRequests = nil

			go func() {
				io.Copy(clientChannel, channel)
				clientChannel.CloseWrite()
			}()
			go func() {
				io.Copy(channel, clientChannel)
				channel.CloseWrite()
			}()

			for {
				select {
				case req, ok := <-requests:
					if !ok {
						// Upstream client closed connection.
						return
					}

					ok, err := clientChannel.SendRequest(req.Type, req.WantReply, req.Payload)
					if err != nil {
						logger("Error while forwarding request: %v", err)
						return
					}

					if req.WantReply {
						req.Reply(ok, nil)
					}

				case req, ok := <-clientChannelReqs:
					if !ok {
						// Downstream client closed connection.
						return
					}

					ok, err := channel.SendRequest(req.Type, req.WantReply, req.Payload)
					if err != nil {
						logger("Error while forwarding client request: %v", err)
						return
					}

					if req.WantReply {
						req.Reply(ok, nil)
					}
				}
			}
		}()
	}
}
