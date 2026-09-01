package main

import (
	"fmt"

	"modelmesh/pkg/log"

	"modelmesh/api"
)

// TODO... make an admin cli
func main() {
	relayAddress := "10.0.0.1""
	relayToken := "123456789"

	c, err := api.NewClient(fmt.Sprintf("http://%s:4002", relayAddress), relayToken)
	if err != nil {
		log.Fatalf("%v", err)
	}

	nodes, err := c.GetPeers()
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("Nodes")
	for _, node := range nodes {
		fmt.Printf("  Node: %s\n", node.Name)
		fmt.Printf("    Mesh Address: %s\n", node.PeerId)
		fmt.Printf("    Update:       %v\n", node.LastUpdate)
	}
}
