package main

import (
	"fmt"

	"modelmesh/pkg/log"

	"modelmesh/pkg/admin"
	"modelmesh/pkg/core"
	"modelmesh/pkg/mesh"
)

func runAdminAndRelay(config *core.Config) error {
	// Load (or create a new) the node identity, identity will persist in this file
	key, err := mesh.LoadOrCreateKey("relay.key")
	if err != nil {
		return err
	}

	discoveredPublicAddress := discoverPublicAddress(config)
	if discoveredPublicAddress == "" {
		log.Errorf("Could  discover a public address. Please set public_address to the ip address of the machine.\n")
		log.Errorf("  - A public address is required to run an relay node\n")
		return nil
	}

	// Init a new admin server... the admin server controls who gets access to our mesh, only nodes with the AdminKey can gain access
	adminSvc, err := admin.NewServer(fmt.Sprintf(":%d", config.Mesh.AdminPort), config.Mesh.AdminKey)
	if err != nil {
		log.Fatalf("Could not initialize relay service. Err:%v\n", err)
	}

	// Initialize a Relay node, this node does not service any application traffic
	//  we gate access using the admin service AllowList as the gatekeeper
	relaySvc, err := mesh.NewRelay(key, []string{discoveredPublicAddress}, mesh.NewGateKeeper(adminSvc.GetAllowList()), config.Mesh.RelayPort)
	if err != nil {
		log.Fatalf("Could not initialize relay service. Err:%v\n", err)
	}

	// Admin service needs to expose the relay addresses for incoming clients.
	//  These addresses are how our peer nodes bootstrap teh p2p network
	adminSvc.WithRelayAddresses(relaySvc.GetAddresses())

	// Run all of our services
	return core.RunInterruptible(adminSvc, relaySvc)
}
