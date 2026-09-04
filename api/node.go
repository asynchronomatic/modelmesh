package api

import (
	"time"

	"github.com/ollama/ollama/api"

	"modelmesh/pkg/core"
)

type ExportedModel struct {
	Name       string // Model Alias
	Model      string // Actual Model name
	Properties api.ListModelResponse
	Process    api.ProcessModelResponse
}

type Node = core.PeerNode

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
