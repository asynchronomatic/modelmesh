package api

import (
	"time"

	"github.com/ollama/ollama/api"
)

type ExportedModel struct {
	Name       string // Model Alias
	Model      string // Actual Model name
	Properties api.ListModelResponse
	Process    api.ProcessModelResponse
}

type Node struct {
	Name        string
	PeerId      string
	LogicalTime uint64
	LastUpdate  time.Time
}

type RegisterNodeRequest struct {
	Node        Node
	Token       string
	LogicalTime uint64
	LastUpdate  time.Time
}

type ListNodesResponse struct {
	Nodes []Node
}

type GetRelayResponse struct {
	ID           string
	MultiAddress []string // the relay multiaddress
	LastUpdate   time.Time
	LogicalTime  uint64
}
