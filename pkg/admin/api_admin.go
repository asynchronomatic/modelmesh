package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/negrel/assert"

	"modelmesh/api"
	"modelmesh/pkg/jsonkv"
)

type inviteSecret struct {
	UUID     string
	MeshId   string
	OneTime  bool
	Expires  int64
	InviteAs string
}

// workflow:
//  create an invite link
//  invite link is sent to a user
//  user passes invite link to ./mesh join
//  ./mesh join does http.get(link)
//  receives the node id, the server config, and a private token for logins
//  writes confi

func newInviteID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func invitePublicID(uuid string) string {
	sum := sha256.Sum256([]byte(uuid))
	return hex.EncodeToString(sum[:])
}

func inviteKVKey(meshID, inviteID string) string {
	return "/invites/" + meshID + "/" + inviteID
}

type meshNodeRecord struct {
	NodeID    string
	Name      string
	AddedAt   time.Time
	InvitedAs string
}

func meshNodeKVKey(meshID, nodeID string) string {
	return "/mesh/" + meshID + "/nodes/" + nodeID
}

func (s *Server) adminCreateInviteLink(ctx *JsonRPC) error {
	assert.Equal("admin", ctx.Group())

	req := api.CreateInviteRequest{}
	if err := ctx.GetObject(&req); err != nil {
		return err
	}
	if req.MeshId == "" {
		return api.NewError(http.StatusBadRequest, "mesh id is required")
	}

	// FIXME: we only have the default mesh
	req.MeshId = "default"

	// make a unique invite code that can be used to jin the mesh
	// 1. The code will expire once older then LifetimeSec
	// 2. If OneTime it should expire after first use
	// 3. Otherwise live forever
	// 4. the code can be stored iun our kv server under /invites/<meshID>/<code>
	inviteUUID, err := newInviteID()
	if err != nil {
		return err
	}

	invite := inviteSecret{
		UUID:     inviteUUID,
		OneTime:  req.OneTime,
		MeshId:   req.MeshId,
		InviteAs: req.Name,
	}
	if req.LifetimeSec != 0 {
		invite.Expires = time.Now().Add(time.Duration(req.LifetimeSec) * time.Second).Unix()
	}

	inviteID := invitePublicID(inviteUUID)
	if err := s.kv.Put(inviteKVKey(req.MeshId, inviteID), invite); err != nil {
		return err
	}

	base := strings.TrimRight(s.baseUrl, "/")
	resp := api.CreateInviteResponse{
		InviteId:   inviteID,
		InviteLink: fmt.Sprintf("%s/redeem/%s", base, inviteID),
	}
	return ctx.ReplyObject(&resp)
}

func (s *Server) adminRedeemInviteLink(ctx *JsonRPC) error {
	inviteID := ctx.PathVar("id")
	if inviteID == "" {
		return api.NewError(http.StatusBadRequest, "invite id is required")
	}

	// FIXME: we only have the default mesh
	key := inviteKVKey("default", inviteID)
	var invite inviteSecret
	if err := s.kv.Get(key, &invite); err != nil {
		if errors.Is(err, jsonkv.ErrNotFound) {
			return api.NewError(http.StatusNotFound, "invite not found")
		}
		return err
	}

	if invite.Expires != 0 && time.Now().Unix() >= invite.Expires {
		_ = s.kv.Delete(key)
		return api.NewError(http.StatusGone, "invite expired")
	}

	req := api.RedeemInviteRequest{}
	if err := ctx.GetObject(&req); err != nil {
		return err
	}
	if req.Node.ID == "" {
		return api.NewError(http.StatusBadRequest, "node peer id is required")
	}
	if req.Node.Name == "" {
		req.Node.Name = invite.InviteAs
	}

	s.acl.Add(req.Node.ID)

	if err := s.kv.Put(meshNodeKVKey(invite.MeshId, req.Node.ID), meshNodeRecord{
		NodeID:    req.Node.ID,
		Name:      req.Node.Name,
		AddedAt:   time.Now().UTC(),
		InvitedAs: invite.InviteAs,
	}); err != nil {
		return err
	}

	if invite.OneTime {
		if err := s.kv.Delete(key); err != nil {
			return err
		}
	}

	resp := api.RedeemInviteResponse{
		MeshId:      invite.MeshId,
		MeshSecret:  s.adminKey,
		MeshServers: append([]string(nil), s.relayAddress...),
	}
	return ctx.ReplyObject(&resp)
}

func (s *Server) adminDeleteInviteLink(ctx *JsonRPC) error {
	assert.Equal("admin", ctx.Group())

	inviteID := ctx.PathVar("id")
	if inviteID == "" {
		return api.NewError(http.StatusBadRequest, "invite id is required")
	}

	// FIXME: we only have the default mesh
	key := inviteKVKey("default", inviteID)
	var invite inviteSecret
	if err := s.kv.Get(key, &invite); err != nil {
		if errors.Is(err, jsonkv.ErrNotFound) {
			return api.NewError(http.StatusNotFound, "invite not found")
		}
		return err
	}

	if err := s.kv.Delete(key); err != nil {
		return err
	}

	return ctx.ReplyObject(&api.DeleteInviteRequest{Invite: inviteID})
}
