package jsonclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Transport interface {
	Get(path string, v interface{}) error
	Post(path string, v interface{}, r interface{}) error
	Put(path string, v interface{}, r interface{}) error
	Delete(path string) error
}

var ErrClientRequest = 800

type Client struct {
	Address string
	token   string

	httpClient *http.Client
	opts       map[string]string
}

func (c *Client) newRequest(method, location string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, location, body)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}
}

func (c *Client) Do(method, location string, in any, out any) error {
	var err error
	client := c.httpClient

	location = fmt.Sprintf("%s%s", c.Address, location)

	var bodyIn io.Reader

	data := []byte("")
	if in != nil {
		data, err = json.Marshal(in)
		if err != nil {
			return err
		}
		bodyIn = bytes.NewBuffer(data)
	}

	req, err := c.newRequest(method, location, bodyIn)
	if err != nil {
		return NewRequestError(ErrClientRequest, err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", resp.Status, string(data))
	}

	if out != nil {
		return json.Unmarshal(data, out)
	}

	return nil
}

func (c *Client) Delete(location string) error {
	return c.Do(http.MethodDelete, location, nil, nil)
}

func (c *Client) Get(location string, v interface{}) error {
	return c.Do(http.MethodGet, location, nil, v)
}

func (c *Client) Post(location string, in interface{}, out interface{}) error {
	return c.Do(http.MethodPost, location, in, out)
}

func (c *Client) Put(location string, in interface{}, out interface{}) error {
	return c.Do(http.MethodPut, location, in, out)
}

func (c *Client) SetOpt(key, value string) {
	c.opts[key] = value
}

func (c *Client) GetOpt(key string) string {
	return c.opts[key]
}

func (c *Client) Close() error {
	c.token = ""
	return nil
}

func (c *Client) WithHttpClient(client *http.Client) *Client {
	c.httpClient = client
	return c
}

func NewClient(address, token string) *Client {
	c := &Client{
		Address:    address,
		token:      token,
		opts:       make(map[string]string),
		httpClient: &http.Client{},
	}

	return c
}
