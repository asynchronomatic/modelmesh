package testable

import (
	"maps"
	"net/http"
	"sync"

	"modelmesh/pkg/core"
	"modelmesh/pkg/jsonclient"
	"modelmesh/pkg/log"
)

// MeshOrchestrator implements the "notion of the p2p network"  it creates a fake P2P network that satisfies the interfaces
// of the proxy.  This allows us to test the proxy in an environment where there is no actual P2P network running
type MeshOrchestrator struct {
	nodes map[string]*MeshNode
	lock  sync.Mutex
}

func (t *MeshOrchestrator) Connect(mn *MeshNode) error {
	t.lock.Lock()
	t.nodes[mn.Node().ID] = mn
	updateNodes := maps.Clone(t.nodes)
	t.lock.Unlock()

	for _, un := range updateNodes {
		un.onNodeConnected(mn.Node())
	}

	return nil
}

func (t *MeshOrchestrator) Disconnect(mn *MeshNode) error {
	t.lock.Lock()
	updateNodes := maps.Clone(t.nodes)
	delete(t.nodes, mn.Node().ID)
	t.lock.Unlock()

	for _, un := range updateNodes {
		un.onNodeDisconnected(mn.Node())
	}

	return nil
}

func (t *MeshOrchestrator) ClientForPeer(dest core.PeerNode, longLived bool) jsonclient.Doer {
	t.lock.Lock()
	node, ok := t.nodes[dest.ID]
	t.lock.Unlock()
	if !ok {
		log.Warnf("no node found for peer %s", dest.ID)
		return &Doer{
			dest:    node.Node(),
			Handler: nil, // will fail to dial
		}
	}

	return &Doer{
		dest:    node.Node(),
		Handler: node.httpHandler,
	}
}

func (t *MeshOrchestrator) ProxyToNode(dest core.PeerNode, w http.ResponseWriter, r *http.Request) {
	t.lock.Lock()
	node, ok := t.nodes[dest.ID]
	t.lock.Unlock()
	if !ok {
		http.Error(w, "dial failure", http.StatusServiceUnavailable)
		return
	}

	node.httpHandler.ServeHTTP(w, r)
}

func (t *MeshOrchestrator) GetPeerMap() (map[string]core.PeerNode, error) {
	t.lock.Lock()
	updateNodes := maps.Clone(t.nodes)
	t.lock.Unlock()

	peerMap := make(map[string]core.PeerNode)
	for _, un := range updateNodes {
		peerMap[un.Node().ID] = un.Node()
	}
	return peerMap, nil
}

func (t *MeshOrchestrator) GetPeerMeshInfo(peer core.PeerNode) *core.MeshInfo {
	info := &core.MeshInfo{
		AdvertisedAddresses: make([]string, 0),
		Connections:         nil,
	}

	t.lock.Lock()
	mn, ok := t.nodes[peer.ID]
	t.lock.Unlock()
	if !ok {
		return info
	}

	info.AdvertisedAddresses = append(info.AdvertisedAddresses, mn.node.ID)
	return info
}

func (t *MeshOrchestrator) NewMeshNode(id string, name string) *MeshNode {
	return &MeshNode{
		node:         core.NewPeerNode(id, name),
		orchestrator: t,
	}
}

func NewMeshOrchestrator() *MeshOrchestrator {
	return &MeshOrchestrator{
		nodes: make(map[string]*MeshNode),
	}
}
