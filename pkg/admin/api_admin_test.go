package admin

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"modelmesh/api"
	"modelmesh/pkg/jsonkv"
)

func newAdminTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	t.Setenv("ADMIN_DB_PATH", filepath.Join(t.TempDir(), "admin.jkv"))
	s, err := NewServer(":0", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.WithBaseUrl("https://mesh.example:4002")
	s.WithRelayAddresses([]string{"/ip4/1.2.3.4/tcp/4001/p2p/relay"})
	ts := httptest.NewServer(s.routes())
	t.Cleanup(ts.Close)
	return s, ts
}

func postJSON(t *testing.T, ts *httptest.Server, method, path, token string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, r)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func createInvite(t *testing.T, ts *httptest.Server, req api.CreateInviteRequest) api.CreateInviteResponse {
	t.Helper()
	res := postJSON(t, ts, http.MethodPost, "/api/v1/admin/invite", "test-secret", req)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create invite: got %d", res.StatusCode)
	}
	var resp api.CreateInviteResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.InviteId == "" {
		t.Fatal("expected invite id")
	}
	if len(resp.InviteId) != 64 {
		t.Fatalf("invite id length %d, want 64-char sha256 hex: %q", len(resp.InviteId), resp.InviteId)
	}
	if _, err := hex.DecodeString(resp.InviteId); err != nil {
		t.Fatalf("invite id is not hex: %q (%v)", resp.InviteId, err)
	}
	return resp
}

func decodeRedeem(t *testing.T, res *http.Response) api.RedeemInviteResponse {
	t.Helper()
	defer res.Body.Close()
	var resp api.RedeemInviteResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestAdminCreateInviteLink(t *testing.T) {
	s, ts := newAdminTestServer(t)

	resp := createInvite(t, ts, api.CreateInviteRequest{
		MeshId:      "mesh-1",
		Name:        "guest",
		OneTime:     true,
		LifetimeSec: 3600,
	})
	wantLink := "https://mesh.example:4002/redeem/" + resp.InviteId
	if resp.InviteLink != wantLink {
		t.Fatalf("InviteLink=%q want %q", resp.InviteLink, wantLink)
	}

	var stored inviteSecret
	if err := s.kv.Get(inviteKVKey("default", resp.InviteId), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.UUID == "" || stored.MeshId != "default" || !stored.OneTime || stored.InviteAs != "guest" {
		t.Fatalf("stored %+v", stored)
	}
	if stored.Expires <= time.Now().Unix() {
		t.Fatalf("expires %d should be in the future", stored.Expires)
	}
}

func TestAdminCreateInviteLinkForever(t *testing.T) {
	s, ts := newAdminTestServer(t)

	resp := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})
	var stored inviteSecret
	if err := s.kv.Get(inviteKVKey("default", resp.InviteId), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Expires != 0 || stored.OneTime {
		t.Fatalf("forever invite %+v", stored)
	}
	if !strings.HasSuffix(resp.InviteLink, "/redeem/"+resp.InviteId) {
		t.Fatalf("InviteLink=%q", resp.InviteLink)
	}
}

func TestAdminCreateInviteLinkRequiresMeshID(t *testing.T) {
	_, ts := newAdminTestServer(t)
	res := postJSON(t, ts, http.MethodPost, "/api/v1/admin/invite", "test-secret", api.CreateInviteRequest{})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing mesh id: got %d", res.StatusCode)
	}
}

func TestAdminRedeemInviteLink(t *testing.T) {
	s, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1", Name: "guest"})

	res := postJSON(t, ts, http.MethodPost, "/redeem/"+created.InviteId, "", api.RedeemInviteRequest{
		Node: api.Node{ID: "peer-redeem-1"},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("redeem: got %d", res.StatusCode)
	}
	resp := decodeRedeem(t, res)
	if resp.MeshId != "default" {
		t.Fatalf("MeshId=%q", resp.MeshId)
	}
	if resp.MeshSecret != "test-secret" {
		t.Fatalf("MeshSecret=%q", resp.MeshSecret)
	}
	if len(resp.MeshServers) != 1 || resp.MeshServers[0] != "/ip4/1.2.3.4/tcp/4001/p2p/relay" {
		t.Fatalf("MeshServers=%v", resp.MeshServers)
	}
	if !s.acl.Has("peer-redeem-1") {
		t.Fatal("expected node to be authorized")
	}

	var rec meshNodeRecord
	if err := s.kv.Get(meshNodeKVKey("default", "peer-redeem-1"), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.NodeID != "peer-redeem-1" || rec.Name != "guest" || rec.InvitedAs != "guest" {
		t.Fatalf("stored node %+v", rec)
	}
	if rec.AddedAt.IsZero() {
		t.Fatal("expected AddedAt")
	}

	res = postJSON(t, ts, http.MethodPost, "/redeem/"+created.InviteId, "", api.RedeemInviteRequest{
		Node: api.Node{ID: "peer-redeem-2"},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reuse forever invite: got %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestAdminRedeemOneTimeInvite(t *testing.T) {
	s, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1", OneTime: true})

	res := postJSON(t, ts, http.MethodPost, "/redeem/"+created.InviteId, "", api.RedeemInviteRequest{
		Node: api.Node{ID: "peer-once-1", Name: "n1"},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("first redeem: got %d", res.StatusCode)
	}
	res.Body.Close()
	if !s.acl.Has("peer-once-1") {
		t.Fatal("expected node to be authorized")
	}
	var rec meshNodeRecord
	if err := s.kv.Get(meshNodeKVKey("default", "peer-once-1"), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.NodeID != "peer-once-1" || rec.Name != "n1" || rec.InvitedAs != "" {
		t.Fatalf("stored node %+v", rec)
	}

	var stored inviteSecret
	if err := s.kv.Get(inviteKVKey("default", created.InviteId), &stored); !errors.Is(err, jsonkv.ErrNotFound) {
		t.Fatalf("one-time invite still stored: %v %+v", err, stored)
	}

	res = postJSON(t, ts, http.MethodPost, "/redeem/"+created.InviteId, "", api.RedeemInviteRequest{
		Node: api.Node{ID: "peer-once-2", Name: "n2"},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("second redeem: got %d want 404", res.StatusCode)
	}
	if s.acl.Has("peer-once-2") {
		t.Fatal("second redeem should not authorize")
	}
}

func TestAdminRedeemExpiredInvite(t *testing.T) {
	s, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1", LifetimeSec: 3600})

	var stored inviteSecret
	key := inviteKVKey("default", created.InviteId)
	if err := s.kv.Get(key, &stored); err != nil {
		t.Fatal(err)
	}
	stored.Expires = time.Now().Add(-time.Second).Unix()
	if err := s.kv.Put(key, stored); err != nil {
		t.Fatal(err)
	}

	res := postJSON(t, ts, http.MethodPost, "/redeem/"+created.InviteId, "", api.RedeemInviteRequest{
		Node: api.Node{ID: "peer-expired-1", Name: "n1"},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusGone {
		t.Fatalf("expired redeem: got %d want 410", res.StatusCode)
	}
	if s.acl.Has("peer-expired-1") {
		t.Fatal("expired invite should not authorize")
	}
	if err := s.kv.Get(key, &stored); !errors.Is(err, jsonkv.ErrNotFound) {
		t.Fatalf("expired invite still stored: %v", err)
	}
}

func TestAdminRedeemInvalidInvite(t *testing.T) {
	_, ts := newAdminTestServer(t)
	res := postJSON(t, ts, http.MethodPost, "/redeem/not-a-valid-token", "", api.RedeemInviteRequest{
		Node: api.Node{ID: "peer-1"},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("invalid invite: got %d want 404", res.StatusCode)
	}
}

func TestRedeemInviteClient(t *testing.T) {
	s, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})

	resp, err := api.RedeemInvite(ts.URL+"/redeem/"+created.InviteId, api.Node{ID: "peer-client-1", Name: "n1"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.MeshId != "default" || resp.MeshSecret != "test-secret" {
		t.Fatalf("response %+v", resp)
	}
	if !s.acl.Has("peer-client-1") {
		t.Fatal("expected node to be authorized")
	}
}

func TestAdminDeleteInviteLink(t *testing.T) {
	s, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})

	res := postJSON(t, ts, http.MethodDelete, "/api/v1/admin/invite/"+created.InviteId, "test-secret", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete invite: got %d", res.StatusCode)
	}

	var stored inviteSecret
	if err := s.kv.Get(inviteKVKey("default", created.InviteId), &stored); !errors.Is(err, jsonkv.ErrNotFound) {
		t.Fatalf("invite still stored: %v %+v", err, stored)
	}

	res2 := postJSON(t, ts, http.MethodPost, "/redeem/"+created.InviteId, "", api.RedeemInviteRequest{
		Node: api.Node{ID: "peer-deleted-1"},
	})
	res2.Body.Close()
	if res2.StatusCode != http.StatusNotFound {
		t.Fatalf("redeem after delete: got %d want 404", res2.StatusCode)
	}
}

func TestAdminDeleteInviteLinkIdempotent(t *testing.T) {
	_, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})

	res := postJSON(t, ts, http.MethodDelete, "/api/v1/admin/invite/"+created.InviteId, "test-secret", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("first delete: got %d", res.StatusCode)
	}

	res = postJSON(t, ts, http.MethodDelete, "/api/v1/admin/invite/"+created.InviteId, "test-secret", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete: got %d want 404", res.StatusCode)
	}
}

func TestAdminDeleteInviteLinkInvalid(t *testing.T) {
	_, ts := newAdminTestServer(t)
	res := postJSON(t, ts, http.MethodDelete, "/api/v1/admin/invite/not-a-valid-token", "test-secret", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("invalid delete: got %d want 404", res.StatusCode)
	}
}

func TestAdminDeleteInviteLinkRequiresAuth(t *testing.T) {
	_, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})
	res := postJSON(t, ts, http.MethodDelete, "/api/v1/admin/invite/"+created.InviteId, "", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated delete: got %d want 401", res.StatusCode)
	}
}

func TestAdminDeleteInviteClient(t *testing.T) {
	s, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})

	client := api.NewClient(ts.URL, "test-secret")
	if err := client.Admin().DeleteInvite(created.InviteId); err != nil {
		t.Fatal(err)
	}

	var stored inviteSecret
	if err := s.kv.Get(inviteKVKey("default", created.InviteId), &stored); !errors.Is(err, jsonkv.ErrNotFound) {
		t.Fatalf("invite still stored: %v", err)
	}
}

func TestAdminRedeemRequiresNodeID(t *testing.T) {
	_, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})
	res := postJSON(t, ts, http.MethodPost, "/redeem/"+created.InviteId, "", api.RedeemInviteRequest{})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing node id: got %d want 400", res.StatusCode)
	}
}
