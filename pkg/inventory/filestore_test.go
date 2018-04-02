package inventory

import (
	"testing"
	"time"
)

func TestFileInventory(t *testing.T) {
	f, _ := NewFileStore("../../test/data/inventory")
	nodes, err := f.Nodes()
	if err != nil {
		t.Fatalf("Unable to read inventory nodes: %s", err)
	}
	t.Logf("Nodes: %s", nodes)
	if len(nodes) != 3 {
		t.Fatalf("The wrong number of nodes were returned: %d", len(nodes))
	}

	t.Logf("Nodes: %s", nodes)
}

func TestRenderedInventoryNode(t *testing.T) {
	f, _ := NewFileStore("../../test/data/inventory")
	nodes, err := f.Nodes()
	if err != nil {
		t.Fatalf("Unable to read inventory nodes: %s", err)
	}
	productionUrl := "http://localhost:8091/master"
	n1, ok := nodes["node0001"]
	if !ok {
		t.Fatalf("Node not found.")
	}

	if n1.Environment == nil {
		t.Fatalf("Environment not set.")
	}

	if n1.Environment.IPXEUrl != productionUrl {
		t.Fatalf("Wrong url parsed for node0001: '%s'", n1.Environment.IPXEUrl)
	}

	if n1.LastUpdated == time.Unix(0, 0) {
		t.Fatalf("Last updated time not populated properly.")
	}

	if n1.Networks["provision"].Network.MTU != 1500 {
		t.Fatalf("MTU not set properly for provisioning network: %v", n1.Networks["provision"].Network.MTU)
	}
}
