package api

import (
	"fmt"
	"time"

	"modelmesh/pkg/jsonclient"
)

type AdminClient struct {
	transport jsonclient.Transport
}

type CreateInviteRequest struct {
	LifetimeSec uint64
	OneTime     bool
	MeshId      string
}

type CreateInviteResponse struct {
	InviteId   string
	InviteLink string
}

func (c *AdminClient) InviteLink(meshId string, lifetime time.Duration) (string, string, error) {
	req := CreateInviteRequest{
		MeshId:      meshId,
		LifetimeSec: uint64(lifetime.Seconds()),
	}
	resp := CreateInviteResponse{}

	err := c.transport.Post("/api/v1/admin/link", &req, &resp)
	if err != nil {
		return "", "", err
	}
	return resp.InviteId, resp.InviteLink, nil
}

type DeleteInviteRequest struct {
	Invite string
}

func (c *AdminClient) DeleteInvite(inviteId string) error {
	return c.transport.Delete(fmt.Sprintf("/api/v1/admin/link/%s", inviteId))
}

func (c *AdminClient) KickPeer(peerId string) error {
	return c.transport.Delete(fmt.Sprintf("/api/v1/admin/peer/%s", peerId))
}
