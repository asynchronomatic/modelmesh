package api

import (
	"fmt"
	"strings"
	"time"

	"modelmesh/pkg/jsonclient"
)

// Client allows for access to the relay control server for managing mes ACLs
type Client struct {
	transport jsonclient.Transport
}

type Registration struct {
	transport jsonclient.Transport
	node      Node
	token     string
}

type RegisterNodeResponse = RegisterNodeRequest

func (r *Registration) Refresh() (bool, uint64, error) {
	req := RegisterNodeRequest{
		Node:  r.node,
		Token: r.token,
	}

	resp := RegisterNodeResponse{}

	// refreshes the node
	err := r.transport.Post(fmt.Sprintf("/api/v1/nodes/%s", r.node.PeerId), &req, &resp)
	if err != nil {
		if strings.Contains(err.Error(), "409") {
			return false, 0, nil
		}
		return false, 0, err
	}
	r.token = resp.Token
	return true, resp.LogicalTime, nil
}

// GetPeers returns a list of currently configured peers for our mesh
func (c *Client) GetPeers() ([]Node, error) {
	resp := ListNodesResponse{}

	err := c.transport.Get("/api/v1/nodes", &resp)
	if err != nil {
		return nil, err
	}

	return resp.Nodes, nil
}

// GetRelay returns the p2p relay address for our mesh along with the last modified timestamp
func (c *Client) GetRelay() ([]string, time.Time, uint64, error) {
	resp := GetRelayResponse{}

	err := c.transport.Get("/api/v1/relay", &resp)
	if err != nil {
		return nil, time.Time{}, 0, err
	}

	return resp.MultiAddress, resp.LastUpdate, resp.LogicalTime, nil
}

// GetAddress returns just the Relay p2p address
func (c *Client) GetAddress() ([]string, error) {
	relay, _, _, err := c.GetRelay()
	return relay, err
}

// Authorize this client for access to the mesh
func (c *Client) Authorize(id string) error {
	req := RegisterNodeRequest{
		Node: Node{
			PeerId: id,
		},
	}

	var resp Node
	return c.transport.Post("/api/v1/authorize", &req, &resp)
}

func (c *Client) Register(name string, id string) (*Registration, error) {
	req := RegisterNodeRequest{
		Node: Node{
			Name:   name,
			PeerId: id,
		},
	}

	resp := RegisterNodeResponse{}
	err := c.transport.Post("/api/v1/nodes", &req, &resp)
	if err != nil {
		return nil, err
	}
	return &Registration{
		transport: c.transport,
		node:      resp.Node,
		token:     resp.Token,
	}, nil
}

// NewClient creates a new client for interacting with the mesh relay control server
func NewClient(address, token string) (*Client, error) {
	transport := jsonclient.NewClient(address, token)
	c := &Client{
		transport: transport,
	}
	return c, nil
}
