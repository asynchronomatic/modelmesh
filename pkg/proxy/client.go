package proxy

import (
	"fmt"

	"modelmesh/pkg/core"
	"modelmesh/pkg/jsonclient"
)

// MeshClient for making api calls across the mesh network
type MeshClient struct {
	address string
	client  jsonclient.Doer
}

type MeshListModelsResponse struct {
	Models []ModelRoute `json:"models"`
}

// GetModelsMesh gets models over the mesh ( internal RPC )
func (c *MeshClient) GetModelsMesh() (map[string]ModelRoute, error) {
	resp := MeshListModelsResponse{}

	jc := jsonclient.NewClient(fmt.Sprintf("http://%s.mesh", c.address), "").WithDoer(c.client)
	err := jc.Get("/.mesh/models", &resp)
	if err != nil {
		return nil, err
	}

	models := make(map[string]ModelRoute)

	for _, m := range resp.Models {
		models[m.Name] = m
	}

	return models, nil
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

func (c *MeshClient) GetMeshStatus() (NodeStatus, error) {
	resp := NodeStatusResponse{}

	jc := jsonclient.NewClient(fmt.Sprintf("http://%s.mesh", c.address), "").WithDoer(c.client)
	err := jc.Get("/.mesh/status", &resp)
	return resp.Status, err
}

func (c *MeshClient) GetMeshMembers() ([]NodeStatus, error) {
	resp := MeshMembersResponse{}

	jc := jsonclient.NewClient(fmt.Sprintf("http://%s.mesh", c.address), "").WithDoer(c.client)
	err := jc.Get("/.mesh/members", &resp)
	return resp.Nodes, err
}

// NewMeshClient creates a client to the ollama interface wrapping the api into our model format
// this client can also use a custom http.Client which is connected over our peer network
func NewMeshClient(address string, client jsonclient.Doer) *MeshClient {
	c := &MeshClient{
		address: address,
		client:  client,
	}
	return c
}
