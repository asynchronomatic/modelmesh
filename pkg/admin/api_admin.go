package admin

import (
	"fmt"
	"time"

	"github.com/negrel/assert"

	"modelmesh/api"
	"modelmesh/pkg/admin/magiclink"
)

type inviteSecret struct {
	InviteId    string
	OneTime     bool
	Expires     int64
	MeshNetwork string
}

// workflow:
//  create an invite link
//  invite link is sent to a user
//  user passes invite link to ./mesh join
//  ./mesh join does http.get(link)
//  receives the node id, the server config, and a private token for logins
//  writes confi

func (s *Server) adminCreateInviteLink(ctx *JsonRPC) error {
	assert.Equal("admin", ctx.Group())

	req := api.CreateInviteRequest{}

	if err := ctx.GetObject(&req); err != nil {
		return err
	}

	// make a unique invite code that can be used to jin the mesh
	// 1. The code will expire once older then LifetimeSec
	// 2. If OneTime it should expire after first use
	// 3. Otherwise live forever
	// 4. the code can be stored iun our kv server under /invites/<meshID>/<code>
	inviteId := "" // FIXME create a unique id based on some salt, timestamp

	invite := inviteSecret{
		InviteId:    inviteId,
		OneTime:     req.OneTime,
		MeshNetwork: req.MeshId,
	}

	if req.LifetimeSec != 0 {
		invite.Expires = time.Now().Add(time.Duration(req.LifetimeSec) * time.Second).Unix()
	}

	magicValue, err := magiclink.New(s.magicKey).Encrypt(&invite)
	if err != nil {
		return err
	}

	// store magicValue under /invites/meshID/<magicValue>

	resp := api.CreateInviteResponse{
		InviteId:   "",
		InviteLink: fmt.Sprintf("%s/invite/%s", s.baseUrl, magicValue),
	}
	return ctx.ReplyObject(&resp)
}
