package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/libp2p/go-libp2p/core/peer"

	"modelmesh/api"
	"modelmesh/pkg/core"
	"modelmesh/pkg/log"
	"modelmesh/pkg/mesh"
)

func runJoin(args []string) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("usage: %s join <invite-url>", os.Args[0])
	}
	link := strings.TrimSpace(args[0])

	key, err := mesh.LoadOrCreateKey(defaultNodeKeyPath)
	if err != nil {
		return fmt.Errorf("node key: %w", err)
	}
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return fmt.Errorf("peer id: %w", err)
	}

	name, _ := os.Hostname()
	resp, err := api.RedeemInvite(link, api.Node{ID: id.String(), Name: name})
	if err != nil {
		return fmt.Errorf("redeem invite: %w", err)
	}

	cfg := configFromInvite(resp)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(defaultConfigPath, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", defaultConfigPath, err)
	}

	log.Infof("wrote %s\n", defaultConfigPath)
	fmt.Println()
	fmt.Println("Joined mesh.")
	fmt.Printf("  config:   %s\n", defaultConfigPath)
	fmt.Printf("  peer id:  %s\n", id)
	fmt.Printf("  mesh id:  %s\n", cfg.Mesh.MeshId)
	fmt.Printf("  address:  %s\n", cfg.Mesh.Address)
	fmt.Println()
	fmt.Println("Next:")
	fmt.Println("  mesh proxy    # start the local proxy on this mesh")

	config := core.MustLoadConfig()
	return runProxy(config)
}

func configFromInvite(resp *api.RedeemInviteResponse) *core.Config {
	cfg := &core.Config{}
	cfg.Proxy.Listen = core.DefaultProxyListen
	cfg.Admin.AdminPort = core.DefaultAdminPort
	cfg.Admin.RelayPort = core.DefaultRelayPort
	cfg.Admin.PublicAddress = "auto"
	cfg.Mesh.Address = strings.TrimSpace(resp.MeshServer)
	cfg.Mesh.Secret = resp.MeshSecret
	cfg.Mesh.MeshId = resp.MeshId
	cfg.Mesh.PublicAddress = "auto"
	cfg.Mesh.Port = 0
	cfg.Mesh.ForcePrivate = false
	cfg.Mesh.MDNSEnabled = true
	cfg.Mesh.Name, _ = os.Hostname()
	cfg.Providers = []core.Provider{{
		ID:        "localhost",
		Type:      "ollama",
		BaseURL:   "http://127.0.0.1:11434",
		Discovery: "pinned",
	}}
	return cfg
}
