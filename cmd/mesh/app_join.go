package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"charm.land/huh/v2"
	"github.com/libp2p/go-libp2p/core/peer"

	"modelmesh/pkg/core"
	"modelmesh/pkg/mesh"
)

func patchYAMLQuoted(src, key, value string) string {
	re := regexp.MustCompile(`(?m)^([ \t]*)` + regexp.QuoteMeta(key) + `:\s*.*$`)
	line := fmt.Sprintf("${1}%s: %q", key, value)
	if re.MatchString(src) {
		return re.ReplaceAllString(src, line)
	}
	return src
}

func writeJoinConfig(path, adminAddress, adminSecret string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out := patchYAMLQuoted(string(data), "admin_address", adminAddress)
	out = patchYAMLQuoted(out, "admin_secret", adminSecret)
	return os.WriteFile(path, []byte(out), 0o600)
}

func runJoin() error {
	if !fileExists(defaultConfigPath) {
		return fmt.Errorf("no %s found; run `mesh init` first", defaultConfigPath)
	}

	config, err := core.LoadConfig()
	if err != nil {
		return fmt.Errorf("load %s: %w", defaultConfigPath, err)
	}

	adminAddress := config.Mesh.AdminAddress
	adminSecret := config.Mesh.AdminKey
	confirm := true

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Admin address").
				Description("HTTP(S) URL of the existing mesh admin. That node must be public or port-forwarded so this machine can reach it.").
				Placeholder(defaultAdminAddress).
				Value(&adminAddress).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().
				Title("Admin secret").
				Description("Shared admin secret for that mesh. Ask the operator of the admin node.").
				Placeholder("required").
				EchoMode(huh.EchoModePassword).
				Value(&adminSecret).
				Validate(huh.ValidateNotEmpty()),
			huh.NewConfirm().
				Title("Join this mesh?").
				Description("Updates admin_address and admin_secret in config.yaml, then authorizes this node's identity with the admin API.").
				Affirmative("Join").
				Negative("Abort").
				Value(&confirm),
		).Title("Join mesh").Description("Connect this node to an existing ModelMesh admin/relay."),
	).WithAccessible(os.Getenv("ACCESSIBLE") != "")

	if err := form.Run(); err != nil {
		if aborted(err) {
			fmt.Println("Aborted.")
			return nil
		}
		return err
	}
	if !confirm {
		fmt.Println("Aborted.")
		return nil
	}

	adminAddress = strings.TrimSpace(adminAddress)
	adminSecret = strings.TrimSpace(adminSecret)

	/*
		client, err := api.NewClient(adminAddress, adminSecret)
		if err != nil {
			return fmt.Errorf("admin client: %w", err)
		}

		addrs, err := client.GetAddress()
		if err != nil {
			return fmt.Errorf("could not reach admin at %s (check address and secret): %w", adminAddress, err)
		}

		if err := writeJoinConfig(defaultConfigPath, adminAddress, adminSecret); err != nil {
			return fmt.Errorf("update %s: %w", defaultConfigPath, err)
		}
		log.Infof("updated %s\n", defaultConfigPath)

		key, err := mesh.LoadOrCreateKey(defaultNodeKeyPath)
		if err != nil {
			return fmt.Errorf("node key: %w", err)
		}
		id, err := peer.IDFromPrivateKey(key)
		if err != nil {
			return fmt.Errorf("peer id: %w", err)
		}

		if err := client.Authorize(id.String()); err != nil {
			return fmt.Errorf("authorize %s: %w", id, err)
		}*/

	key, err := mesh.LoadOrCreateKey(defaultNodeKeyPath)
	if err != nil {
		return fmt.Errorf("node key: %w", err)
	}
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return fmt.Errorf("peer id: %w", err)
	}

	fmt.Println()
	fmt.Println("Joined mesh.")
	fmt.Printf("  admin:    %s\n", adminAddress)
	fmt.Printf("  peer id:  %s\n", id)
	/*
		if len(addrs) > 0 {
			fmt.Println("  relays:")
			for _, a := range addrs {
				fmt.Printf("    %s\n", a)
			}
		}*/
	fmt.Println()
	fmt.Println("Next:")
	fmt.Println("  mesh proxy    # start the local proxy on this mesh")
	return nil
}
