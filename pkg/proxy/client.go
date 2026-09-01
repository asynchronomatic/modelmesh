package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	oapi "github.com/ollama/ollama/api"

	"modelmesh/pkg/log"

	"modelmesh/pkg/core"
	"modelmesh/pkg/jsonclient"
)

// Client for making api calls across the mesh network
type Client struct {
	address string
	client  *http.Client
}

// GetModels returns a list of pinned(exported) models on a given ollama server
func (c *Client) GetModels() (map[string]*ModelRoute, error) {
	newTable := make(map[string]*ModelRoute)

	u, err := url.Parse(fmt.Sprintf("http://%s.mesh", c.address))
	if err != nil {
		log.Fatalf("Failed to parse provider URL: %v", err)
	}

	client := oapi.NewClient(u, c.client)
	ctx := context.Background()

	running, err := client.ListRunning(ctx) // GET /api/ps
	if err != nil {
		return nil, err
	}

	for _, m := range running.Models {
		cfg, ok := newTable[m.Name]
		if !ok {
			cfg = &ModelRoute{}
		}
		cfg.Name = m.Name
		cfg.Model = m.Name
		cfg.Process = m
		newTable[m.Name] = cfg
	}

	available, err := client.List(ctx) // GET /api/tags
	if err != nil {
		return nil, err
	}

	for _, m := range available.Models {
		cfg, ok := newTable[m.Name]
		if !ok {
			continue
		}
		cfg.Properties = m
		newTable[m.Name] = cfg
	}

	return newTable, nil
}

type MeshListModelsResponse struct {
	Models []ModelRoute `json:"models"`
}

// GetModelsMesh gets models over the mesh ( internal RPC )
func (c *Client) GetModelsMesh() (map[string]*ModelRoute, error) {
	resp := MeshListModelsResponse{}

	jc := jsonclient.NewClient(fmt.Sprintf("http://%s.mesh", c.address), "").WithHttpClient(c.client)
	err := jc.Get("/.mesh/models", &resp)
	if err != nil {
		return nil, err
	}

	newTable := make(map[string]*ModelRoute)

	for _, m := range resp.Models {
		cfg, ok := newTable[m.Name]
		if !ok {
			cfg = &ModelRoute{}
		}
		cfg.Name = m.Name
		cfg.Model = m.Model
		cfg.Properties = m.Properties
		cfg.Process = m.Process
		newTable[cfg.Name] = cfg
	}

	return newTable, nil
}

type NodeStatus struct {
	Name      string
	PeerID    string
	Type      string
	Reachable bool
	Models    []string
	Mesh      *core.MeshInfo
}

type NodeStatusResponse struct {
	Status NodeStatus
}

func (c *Client) GetMeshStatus() (NodeStatus, error) {
	resp := NodeStatusResponse{}

	jc := jsonclient.NewClient(fmt.Sprintf("http://%s.mesh", c.address), "").WithHttpClient(c.client)
	err := jc.Get("/.mesh/status", &resp)
	return resp.Status, err
}

// NewClient creates a client to the ollama interface wrapping the api into our model format
// this client can also use a custom http.Client which is connected over our peer network
func NewClient(address string, client *http.Client) *Client {
	c := &Client{
		address: address,
		client:  client,
	}
	return c
}
