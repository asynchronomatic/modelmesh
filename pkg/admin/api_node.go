package admin

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"modelmesh/pkg/log"

	"modelmesh/api"
)

func newNodeToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}

func (s *Server) apiNodeAuthorize(ctx *JsonRPC) error {
	var req api.RegisterNodeRequest
	if err := ctx.GetObject(&req); err != nil {
		return api.NewError(http.StatusBadRequest, err.Error())
	}
	if req.Node.ID == "" {
		return api.NewError(http.StatusBadRequest, "node peer id is required")
	}

	s.acl.Add(req.Node.ID)
	return ctx.ReplyObject(&req.Node)
}

// TODO: nodes need to expire if we have not heard from them in a while
func (s *Server) apiNodeRegister(ctx *JsonRPC) error {
	var req api.RegisterNodeRequest
	if err := ctx.GetObject(&req); err != nil {
		return api.NewError(http.StatusBadRequest, err.Error())
	}
	if req.Node.Name == "" {
		return api.NewError(http.StatusBadRequest, "node name is required")
	}
	if req.Node.ID == "" {
		return api.NewError(http.StatusBadRequest, "node peer id is required")
	}

	if !s.acl.Has(req.Node.ID) {
		return api.NewError(http.StatusBadRequest, "node not Authorized")
	}

	s.lock.Lock()
	s.lastUpdate = time.Now()
	s.logicalTime++

	req.Node.LastUpdate = s.lastUpdate
	req.Node.LogicalTime = s.logicalTime

	resp := api.RegisterNodeRequest{
		Node:        req.Node,
		Token:       newNodeToken(),
		LastUpdate:  s.lastUpdate,
		LogicalTime: s.logicalTime,
	}

	s.nodes[req.Node.ID] = &NodeReference{
		Node:     req.Node,
		LastPing: time.Now(),
		Token:    resp.Token,
	}
	s.lock.Unlock()
	log.Infof("registered node %s", resp.Node.ID)

	return ctx.ReplyObject(&resp)
}

func (s *Server) apiNodeRefresh(ctx *JsonRPC) error {
	id := ctx.PathVar("id")
	if id == "" {
		return api.NewError(http.StatusBadRequest, "node id is required")
	}

	req := api.RegisterNodeRequest{}
	if err := ctx.GetObject(&req); err != nil {
		return api.NewError(http.StatusBadRequest, err.Error())
	}

	if req.Node.ID != id {
		return api.NewError(http.StatusBadRequest, "node id mismatch")
	}

	updateNode := func(req *api.RegisterNodeRequest) bool {
		if req.Token == "" {
			return false
		}

		ref, ok := s.nodes[id]
		if !ok {
			return false
		}

		if ref.Token != req.Token {
			return false
		}
		ref.LastPing = time.Now()
		return true
	}

	resp := api.RegisterNodeRequest{
		Node:  req.Node,
		Token: req.Token,
	}
	s.lock.Lock()
	resp.LogicalTime = s.logicalTime
	resp.LastUpdate = s.lastUpdate
	valid := updateNode(&req)
	s.lock.Unlock()

	if !valid {
		return api.NewError(http.StatusConflict, "node registration invalid")
	}

	return ctx.ReplyObject(&resp)
}

func (s *Server) apiNodeUnregister(ctx *JsonRPC) error {
	id := ctx.PathVar("id")
	if id == "" {
		return api.NewError(http.StatusBadRequest, "node id is required")
	}

	s.lock.Lock()
	node, ok := s.nodes[id]
	if ok {
		s.lastUpdate = time.Now()
		s.logicalTime++
	}
	delete(s.nodes, id)
	s.lock.Unlock()

	s.acl.Remove(id) // can be done outside the lock
	log.Infof("unregistered node %s", id)

	if !ok {
		return api.NewError(http.StatusNotFound, "node not found")
	}
	return ctx.ReplyObject(&node)
}

func (s *Server) apiNodeList(ctx *JsonRPC) error {
	s.lock.Lock()
	resp := api.ListNodesResponse{
		Nodes: make([]api.Node, 0, len(s.nodes)),
	}

	for _, ref := range s.nodes {
		resp.Nodes = append(resp.Nodes, ref.Node)
	}
	s.lock.Unlock()

	return ctx.ReplyObject(&resp)
}

func (s *Server) apiRelayGet(ctx *JsonRPC) error {
	s.lock.Lock()
	resp := api.GetRelayResponse{
		MultiAddress: s.relayAddress,
		LastUpdate:   s.lastUpdate,
		LogicalTime:  s.logicalTime,
	}
	s.lock.Unlock()

	return ctx.ReplyObject(&resp)
}

func (s *Server) apiNodeLogin(ctx *JsonRPC) error {

}
