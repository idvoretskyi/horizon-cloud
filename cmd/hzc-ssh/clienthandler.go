package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"

	"golang.org/x/crypto/ssh"
)

var (
	sshVersionString = "SSH-2.0-HorizonCloudProxy"
)

type clientConn struct {
	sock      net.Conn
	config    *config
	log       *hzlog.Logger
	clientKey string
}

func (c *clientConn) makeServerConfig() *ssh.ServerConfig {
	serverConfig := &ssh.ServerConfig{
		ServerVersion: sshVersionString,
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() != "auth" {
				return nil, errors.New("Username must be 'auth'")
			}

			c.clientKey = base64.StdEncoding.EncodeToString(key.Marshal())
			c.log.Info("key is %s", c.clientKey)

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

	logger.Info("Opened channel")
	defer logger.Info("Done with channel")

	enc := json.NewEncoder(channel)

	resp, err := c.config.APIClient.GetUsersByKey(api.GetUsersByKeyReq{PublicKey: c.clientKey})
	if err != nil {
		logger.Error("Couldn't get users for %v: %v", c.clientKey, err)
		enc.Encode(api.Resp{
			Success: false,
			Error:   "Internal error",
		})
		return
	}

	if len(resp.Users) == 0 {
		enc.Encode(api.Resp{
			Success: false,
			Error:   fmt.Sprintf("No user has your key (%v) attached.", c.clientKey),
		})
		return
	}

	var response struct {
		Token string
	}

	response.Token, err = api.SignToken(&api.TokenData{
		Users: resp.Users,
	}, c.config.TokenSecret)
	if err != nil {
		logger.Error("Couldn't sign token: %v", err)
		enc.Encode(api.Resp{
			Success: false,
			Error:   "Internal error",
		})
		return
	}

	msg, err := json.Marshal(response)
	if err != nil {
		logger.Error("Couldn't marshal response: %v", err)
		enc.Encode(api.Resp{
			Success: false,
			Error:   "Internal error",
		})
		return
	}

	rawMsg := json.RawMessage(msg)

	enc.Encode(api.Resp{
		Success: true,
		Content: &rawMsg,
	})

	channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
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
