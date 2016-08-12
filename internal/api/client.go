package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

func (c *Client) GetUsersByKey(
	opts GetUsersByKeyReq) (*GetUsersByKeyResp, error) {

	var ret GetUsersByKeyResp
	err := c.jsonRoundTrip(GetUsersByKeyPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetProjectAddrsByKey(
	opts GetProjectAddrsByKeyReq) (*GetProjectAddrsByKeyResp, error) {

	var ret GetProjectAddrsByKeyResp
	err := c.jsonRoundTrip(GetProjectAddrsByKeyPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetProjectAddrByDomain(
	opts GetProjectAddrByDomainReq) (*GetProjectAddrByDomainResp, error) {
	var ret GetProjectAddrByDomainResp
	err := c.jsonRoundTrip(GetProjectAddrByDomainPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetProjectsByToken(
	opts GetProjectsByTokenReq) (*GetProjectsByTokenResp, error) {
	var ret GetProjectsByTokenResp
	err := c.jsonRoundTrip(GetProjectsByTokenPath, opts, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) UpdateProjectManifest(
	opts UpdateProjectManifestReq) (*UpdateProjectManifestResp, error) {
	var ret UpdateProjectManifestResp
	err := c.jsonRoundTrip(UpdateProjectManifestPath, opts, &ret)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to decode a JSON error response object out of the response body.

		body, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 1024))
		var errBody struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errBody) == nil {
			return fmt.Errorf("couldn't %v %v: %v",
				req.Method, req.URL, errBody.Error)
		}

		// Couldn't decode the body as JSON; just return a generic HTTP error.
		return fmt.Errorf("couldn't %v %v: response code %v, body %#v",
			req.Method, req.URL, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
