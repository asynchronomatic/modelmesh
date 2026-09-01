package proxy

import (
	"encoding/json"
	"net/http"

	"modelmesh/pkg/log"
)

func (p *Proxy) apiMeshStatus(w http.ResponseWriter, r *http.Request) {
	resp := NodeStatusResponse{
		Status: NodeStatus{
			Name:      p.mesh.Name,
			PeerID:    p.mesh.ID,
			Reachable: true,
			Models:    make([]string, 0),
		},
	}

	for k := range p.localRoutes {
		resp.Status.Models = append(resp.Status.Models, k)
	}
	resp.Status.Mesh = p.mesh.InspectPeer(p.mesh.ID) // inspect self

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&resp)
}

type MeshMembersResponse struct {
	Nodes []NodeStatus
}

func (p *Proxy) apiMeshMembers(w http.ResponseWriter, r *http.Request) {
	resp := MeshMembersResponse{}

	peers, err := p.mesh.GetPeerMap()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, peer := range peers {
		var status NodeStatus
		if peer.PeerId == p.mesh.ID {
			status = NodeStatus{
				Name:      peer.Name,
				PeerID:    p.mesh.ID,
				Reachable: true,
				Type:      "self",
				Models:    make([]string, 0),
			}
			for k := range p.localRoutes {
				status.Models = append(status.Models, k)
			}
			status.Mesh = p.mesh.InspectPeer(peer.PeerId)
		} else {
			client := NewClient(peer.PeerId, p.mesh.ClientForPeer(peer.PeerId))
			status, err = client.GetMeshStatus()
			if err != nil {
				log.WithName("proxy").Infof("failed to get mesh status from peer %s: %v", peer.PeerId, err)
				status.PeerID = peer.PeerId
				status.Name = peer.Name
				status.Reachable = false
			}
			status.Type = "peer"
		}

		if status.Mesh != nil {
			for idx, c := range status.Mesh.Connections {
				if node, ok := peers[c.PeerID]; ok {
					status.Mesh.Connections[idx].PeerName = node.Name
				}
			}
		}
		resp.Nodes = append(resp.Nodes, status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&resp)
}

// apiMeshModels is used to fetch model maps between mesh nodes
func (p *Proxy) apiMeshModels(w http.ResponseWriter, r *http.Request) {
	resp := MeshListModelsResponse{}

	for _, r := range p.localRoutes {
		meshModel := *r
		meshModel.Servers = map[string]string{
			p.mesh.ID: p.mesh.ID,
		}
		resp.Models = append(resp.Models, meshModel)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&resp)
}
