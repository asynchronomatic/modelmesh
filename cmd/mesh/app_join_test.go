package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"modelmesh/api"
	"modelmesh/pkg/core"
)

func TestConfigFromInvite(t *testing.T) {
	cfg := configFromInvite(&api.RedeemInviteResponse{
		MeshId:     "default",
		MeshSecret: "sekrit",
		MeshServer: "http://10.0.0.30:4002",
	})
	if cfg.Mesh.Address != "http://10.0.0.30:4002" {
		t.Fatalf("Address=%q", cfg.Mesh.Address)
	}
	if cfg.Mesh.MeshId != "default" || cfg.Mesh.Secret != "sekrit" {
		t.Fatalf("mesh %+v", cfg.Mesh)
	}
	if cfg.Proxy.Listen != core.DefaultProxyListen {
		t.Fatalf("listen %q", cfg.Proxy.Listen)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].Type != "ollama" {
		t.Fatalf("providers %+v", cfg.Providers)
	}
}

func TestRunJoin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		var req api.RedeemInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode: %v", err)
		}
		if req.Node.ID == "" {
			t.Error("missing node id")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.RedeemInviteResponse{
			MeshId:     "default",
			MeshSecret: "join-secret",
			MeshServer: "http://10.0.0.30:4002",
		})
	}))
	t.Cleanup(ts.Close)

	if err := runJoin([]string{ts.URL + "/redeem/abc"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, defaultConfigPath)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, defaultNodeKeyPath)); err != nil {
		t.Fatal(err)
	}

	cfg, err := core.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mesh.Address != "http://10.0.0.30:4002" || cfg.Mesh.Secret != "join-secret" || cfg.Mesh.MeshId != "default" {
		t.Fatalf("loaded mesh %+v", cfg.Mesh)
	}
}

func TestRunJoinRequiresURL(t *testing.T) {
	if err := runJoin(nil); err == nil {
		t.Fatal("expected usage error")
	}
}
