package proxy

import (
	"net/http"

	"github.com/negrel/assert"

	"github.com/asynchronomatic/speakeasy/api"
	"github.com/asynchronomatic/speakeasy/pkg/log"
)

func (p *Proxy) adminEnabledHandler(rpc *RPC) error {
	assert.NotNil(p.admin)

	resp := struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: true,
	}

	return rpc.ReplyObject(&resp)
}

func (p *Proxy) adminCreateInvitedHandler(rpc *RPC) error {
	assert.NotNil(p.admin)

	req := api.CreateInviteRequest{}
	if err := rpc.GetObject(&req); err != nil {
		return err
	}
	if req.MeshId == "" {
		req.MeshId = "default"
	}
	resp, err := p.admin.CreateInvite(req)
	if err != nil {
		return err
	}
	return rpc.ReplyObject(resp)
}

func (p *Proxy) adminListInvitesHandler(rpc *RPC) error {
	assert.NotNil(p.admin)

	resp, err := p.admin.ListInvites()
	if err != nil {
		log.WithName("proxy").Warnf("Failed to list invites: %v", err)
		return err
	}
	return rpc.ReplyObject(resp)
}

func (p *Proxy) adminRevokeInviteHandler(rpc *RPC) error {
	assert.NotNil(p.admin)

	id := rpc.PathVar("id")
	if id == "" {
		return api.NewError(http.StatusBadRequest, "invite id is required")
	}
	if err := p.admin.DeleteInvite(id); err != nil {
		return err
	}
	return rpc.ReplyObject(&api.DeleteInviteRequest{Invite: id})
}

func (p *Proxy) adminListNodesHandler(rpc *RPC) error {
	assert.NotNil(p.admin)

	resp, err := p.admin.ListNodes()
	if err != nil {
		return err
	}
	return rpc.ReplyObject(resp)
}

func (p *Proxy) adminKickNodeHandler(rpc *RPC) error {
	assert.NotNil(p.admin)

	id := rpc.PathVar("id")
	if id == "" {
		return api.NewError(http.StatusBadRequest, "node id is required")
	}
	if err := p.admin.KickPeer(id); err != nil {
		return err
	}
	return rpc.ReplyObject(&api.KickPeerResponse{NodeID: id})
}
