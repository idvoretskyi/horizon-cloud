package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/rethinkdb/horizon-cloud/internal/types"

	"golang.org/x/crypto/ssh"
)

var (
	projectEnvVarName = []byte(api.ProjectEnvVarName)
	sshVersionString  = "SSH-2.0-HorizonCloudProxy"
)

type clientConn struct {
	sock           net.Conn
	config         *config
	log            *hzlog.Logger
	projectTargets []types.Project
}

func (c *clientConn) makeServerConfig() *ssh.ServerConfig {
	serverConfig := &ssh.ServerConfig{
		ServerVersion: sshVersionString,
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() != "horizon" {
				return nil, errors.New("Username must be 'horizon'")
			}

			c.log.Info("key is %s", base64.StdEncoding.EncodeToString(key.Marshal()))

			resp, err := c.config.APIClient.GetProjectsByKey(api.GetProjectsByKeyReq{
				PublicKey: base64.StdEncoding.EncodeToString(key.Marshal()),
			})
			if err != nil {
				c.log.Error("Couldn't talk to API: %v", err)
				return nil, errors.New("Couldn't talk to API")
			}

			if len(resp.Projects) == 0 {
				return nil, errors.New("Unknown SSH key")
			}

			c.projectTargets = resp.Projects

			return nil, nil
		},
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			if err == nil {
				c.log.Info("Authentication method=%v succeeded", method)
			} else {
				c.log.Info("Authentication method=%v failed: %v", method, err)
			}
		},
	}

	serverConfig.AddHostKey(c.config.HostKey)

	return serverConfig
}

func (c *clientConn) handleSSHChannel(
	channel ssh.Channel,
	requests <-chan *ssh.Request,
	upstreamType string,
	upstreamExtra []byte) {

	defer channel.Close()

	logger := c.log.With(map[string]interface{}{
		"channelid": fmt.Sprintf("%p", channel),
	})

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

			logger.Info("request type=%v, extra=%#v", req.Type, string(req.Payload))

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

	if projectName != "" {
		logger = logger.With(map[string]interface{}{
			"projectname": projectName,
		})
	}

	logger.Info("got projectName = %#v", projectName)

	// Phase 2: Verify that the project name was given and is valid.

	var projectErr error
	var project types.Project
	if projectName == "" {
		projectErr = errors.New("No project name passed")
	} else {
		found := false
		for _, proj := range c.projectTargets {
			if proj.Name == projectName {
				project = proj
				found = true
				break
			}
		}
		if !found {
			projectErr = fmt.Errorf("Project `%v` not found", projectName)
		}
	}

	if projectErr != nil {
		logger.UserError("Couldn't find an appropriate target: %v", projectErr)
		fmt.Fprintf(channel.Stderr(),
			"Couldn't find an appropriate target: %v\n", projectErr)
		return
	}

	// Phase 3: Connect to target.

	logger.Info("Connecting to %v (project name %#v)",
		project.SSHAddress, project.Name)

	clientNet, err := net.Dial("tcp", project.SSHAddress)
	if err != nil {
		logger.Error("Couldn't connect to %v: %v", project.SSHAddress, err)
		fmt.Fprintf(channel.Stderr(),
			"Couldn't connect to server hosting project `%v`\n", project.Name)
		return
	}
	defer clientNet.Close()

	clientConfig := &ssh.ClientConfig{
		User: "horizon",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(c.config.ClientKey),
		},
		ClientVersion: sshVersionString,
	}

	clientConn, clientChans, clientReqs, err :=
		ssh.NewClientConn(clientNet, project.SSHAddress, clientConfig)
	if err != nil {
		logger.Error("Couldn't setup SSH connection to %v: %v", project.SSHAddress, err)
		fmt.Fprintf(channel.Stderr(),
			"Couldn't connect to server hosting project `%v`\n", project.Name)
		return
	}

	go ssh.DiscardRequests(clientReqs)

	go func() {
		for newCh := range clientChans {
			logger.Info("Rejecting upstream channel request")
			newCh.Reject(ssh.Prohibited, "prohibited")
		}
	}()

	logger.Info("Connected to upstream, creating client channel.")

	clientChannel, clientChannelReqs, err := clientConn.OpenChannel(
		upstreamType, upstreamExtra)
	if err != nil {
		logger.Error("Couldn't create client channel: %v", err)
		fmt.Fprintf(channel.Stderr(),
			"Couldn't connect to server hosting project `%v`\n", project.Name)
		return
	}

	// Phase 4: Forward channel requests and data.

	logger.Info("Client channel ready, forwarding requests and data.")

	for _, req := range pendingRequests {
		ok, err := clientChannel.SendRequest(req.Type, req.WantReply, req.Payload)
		if err != nil {
			logger.Error("Error while forwarding pending request: %v", err)
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
				logger.Error("Error while forwarding request: %v", err)
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
				logger.Error("Error while forwarding client request: %v", err)
				return
			}

			if req.WantReply {
				req.Reply(ok, nil)
			}
		}
	}
}

func handleClientConn(baseLogger *hzlog.Logger, sock net.Conn, config *config) {
	logger := baseLogger.With(map[string]interface{}{
		"remoteaddr": sock.RemoteAddr(),
		"localaddr":  sock.LocalAddr(),
	})

	c := &clientConn{
		sock:   sock,
		config: config,
		log:    logger,
	}

	c.log.Info("Accepted new connection")
	defer c.log.Info("Done with connection")
	defer sock.Close()

	serverConfig := c.makeServerConfig()

	serverConn, chans, reqs, err := ssh.NewServerConn(sock, serverConfig)
	if err != nil {
		c.log.UserError("Failed to set up ssh connection: %v", err)
		return
	}

	c.log.Info("Handshake complete, ClientVersion=%#v",
		string(serverConn.ClientVersion()))

	go ssh.DiscardRequests(reqs)

	var wg sync.WaitGroup
	defer wg.Wait()

	for newCh := range chans {
		upstreamType := newCh.ChannelType()
		upstreamExtra := newCh.ExtraData()

		if upstreamType != "session" {
			c.log.Info("Rejecting channel of type %v", newCh.ChannelType())
			newCh.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newCh.Accept()
		if err != nil {
			c.log.UserError("Error accepting new channel: %v", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			c.handleSSHChannel(channel, requests, upstreamType, upstreamExtra)
		}()

	}
}
