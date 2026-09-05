package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"modelmesh/api"
	"modelmesh/pkg/admin/auth"
	"modelmesh/pkg/admin/magiclink"
)

func TestApiNodeLogin(t *testing.T) {
	s, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1", Name: "guest"})
	join, err := api.RedeemInvite(ts.URL+"/redeem/"+created.InviteId, api.Node{ID: "peer-login-1", Name: "n1"})
	if err != nil {
		t.Fatal(err)
	}

	res := postJSON(t, ts, http.MethodPost, "/api/v1/login", "", api.NodeLoginRequest{
		NodeID:     "peer-login-1",
		MeshId:     "default",
		MeshSecret: join.MeshSecret,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login: got %d", res.StatusCode)
	}
	var login api.NodeLoginResponse
	if err := json.NewDecoder(res.Body).Decode(&login); err != nil {
		t.Fatal(err)
	}
	if login.NodeID != "peer-login-1" {
		t.Fatalf("NodeID=%q", login.NodeID)
	}
	if !strings.HasPrefix(login.Token, auth.SessionTokenPrefix) {
		t.Fatalf("token prefix: %q", login.Token)
	}
	if login.Expires != 0 {
		t.Fatalf("expected no expiry, got %d", login.Expires)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	authRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	authRes.Body.Close()
	if authRes.StatusCode != http.StatusOK {
		t.Fatalf("session GET /nodes: got %d", authRes.StatusCode)
	}

	props, err := s.authenticateSessionToken(login.Token)
	if err != nil {
		t.Fatal(err)
	}
	if props.User != "peer-login-1" || props.Group != "mesh" {
		t.Fatalf("session props %+v", props)
	}
}

func TestApiNodeLoginRejectsBadSecret(t *testing.T) {
	_, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})
	if _, err := api.RedeemInvite(ts.URL+"/redeem/"+created.InviteId, api.Node{ID: "peer-login-bad", Name: "n1"}); err != nil {
		t.Fatal(err)
	}

	res := postJSON(t, ts, http.MethodPost, "/api/v1/login", "", api.NodeLoginRequest{
		NodeID:     "peer-login-bad",
		MeshSecret: "wrong-secret",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad secret: got %d want 401", res.StatusCode)
	}
}

func TestApiNodeLoginExpiredSession(t *testing.T) {
	s, ts := newAdminTestServer(t)
	claims := sessionClaims{NodeID: "peer-expired-session", Expires: time.Now().Add(-time.Second).Unix()}
	raw, err := magiclink.New(s.magicKey).Encrypt(&claims)
	if err != nil {
		t.Fatal(err)
	}
	token := auth.SessionTokenPrefix + raw

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired session: got %d want 401", res.StatusCode)
	}
}

func TestApiNodeLoginLifetime(t *testing.T) {
	_, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})
	join, err := api.RedeemInvite(ts.URL+"/redeem/"+created.InviteId, api.Node{ID: "peer-login-ttl", Name: "n1"})
	if err != nil {
		t.Fatal(err)
	}

	res := postJSON(t, ts, http.MethodPost, "/api/v1/login", "", api.NodeLoginRequest{
		NodeID:      "peer-login-ttl",
		MeshSecret:  join.MeshSecret,
		LifetimeSec: 3600,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login: got %d", res.StatusCode)
	}
	var login api.NodeLoginResponse
	if err := json.NewDecoder(res.Body).Decode(&login); err != nil {
		t.Fatal(err)
	}
	if login.Expires <= time.Now().Unix() {
		t.Fatalf("expires %d should be in the future", login.Expires)
	}
}

func TestMeshClientLogin(t *testing.T) {
	_, ts := newAdminTestServer(t)
	created := createInvite(t, ts, api.CreateInviteRequest{MeshId: "mesh-1"})
	join, err := api.RedeemInvite(ts.URL+"/redeem/"+created.InviteId, api.Node{ID: "peer-client-login", Name: "n1"})
	if err != nil {
		t.Fatal(err)
	}

	mc, err := api.NewClient(ts.URL, "").Mesh("default")
	if err != nil {
		t.Fatal(err)
	}
	login, err := mc.Login("peer-client-login", join.MeshSecret)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(login.Token, auth.SessionTokenPrefix) {
		t.Fatalf("token %q", login.Token)
	}
}
