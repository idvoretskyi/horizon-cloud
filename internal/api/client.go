package api

import (
	"bytes"
	"encoding/json"
	"net/http"
)

const (
	jsonMIMEType       = "application/json; charset=utf-8"
	sharedSecretHeader = "X-Horizon-Cloud-Shared-Secret"
)

type Client struct {
	baseURL      string
	sharedSecret string
}

// Constructs a new Client object.
//
// baseURL must be the prefix of the API URL; for example, "https://horizon" if
// the calls should be "https://horizon/v1/configs/...".
//
// sharedSecret should be the shared secret for accessing protected APIs.
func NewClient(baseURL string, sharedSecret string) (*Client, error) {
	return &Client{
		baseURL:      baseURL,
		sharedSecret: sharedSecret,
	}, nil
}

func (c *Client) EnsureConfigConnectable(
	opts EnsureConfigConnectableReq) (*EnsureConfigConnectableResp, error) {

	var ret EnsureConfigConnectableResp
	err := c.jsonRoundTrip(EnsureConfigConnectablePath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetConfig(
	opts GetConfigReq) (*GetConfigResp, error) {

	var ret GetConfigResp
	err := c.jsonRoundTrip(GetConfigPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) WaitConfigApplied(
	opts WaitConfigAppliedReq) (*WaitConfigAppliedResp, error) {

	// RSI: retry requests if they fail (but only for network reasons)

	var ret WaitConfigAppliedResp
	err := c.jsonRoundTrip(WaitConfigAppliedPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetProjectsByKey(
	opts GetProjectsByKeyReq) (*GetProjectsByKeyResp, error) {

	var ret GetProjectsByKeyResp
	err := c.jsonRoundTrip(GetProjectsByKeyPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetProjectByDomain(
	opts GetProjectByDomainReq) (*GetProjectByDomainResp, error) {
	var ret GetProjectByDomainResp
	err := c.jsonRoundTrip(GetProjectByDomainPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) jsonRoundTrip(path string, body interface{}, out interface{}) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", jsonMIMEType)
	if c.sharedSecret != "" {
		req.Header.Set(sharedSecretHeader, c.sharedSecret)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return ReadJSONResp(resp, out)
}
