package api

import (
	"bytes"
	"encoding/json"
	"net/http"
)

const jsonMIMEType = "application/json; charset=utf-8"

type Client struct {
	baseURL string
}

// Constructs a new Client object. baseURL must be the prefix of the API URL;
// for example, "https://fusion" if the calls should be
// "https://fusion/v1/config/..."
func NewClient(baseURL string) (*Client, error) {
	return &Client{
		baseURL: baseURL,
	}, nil
}

func (c *Client) EnsureConfigConnectable(
	opts EnsureConfigConnectableReq) (*EnsureConfigConnectableResp, error) {

	var ret EnsureConfigConnectableResp
	err := c.jsonRoundTrip("/v1/config/ensure_connectable", opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetConfig(
	opts GetConfigReq) (*GetConfigResp, error) {

	var ret GetConfigResp
	err := c.jsonRoundTrip("/v1/config/get", opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) WaitConfigApplied(
	opts WaitConfigAppliedReq) (*WaitConfigAppliedResp, error) {

	// RSI: retry requests if they fail (but only for network reasons)

	var ret WaitConfigAppliedResp
	err := c.jsonRoundTrip("/v1/config/wait_applied", opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetProjects(opts GetProjectsReq) (*GetProjectsResp, error) {
	var ret GetProjectsResp
	err := c.jsonRoundTrip("/v1/projects/get", opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetByAlias(opts GetByAliasReq) (*GetByAliasResp, error) {
	var ret GetByAliasResp
	err := c.jsonRoundTrip("/v1/projects/getByAlias", opts, &ret)
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

	resp, err := http.Post(c.baseURL+path, jsonMIMEType, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return ReadJSONResp(resp, out)
}
