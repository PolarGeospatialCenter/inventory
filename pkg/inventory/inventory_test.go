package inventory

import (
	"testing"
)

func TestCopyNode(t *testing.T) {
	src, err := NewSampleInventoryStore()
	if err != nil {
		t.Fatalf("Unable to populate source data store with sample data")
	}

	dst := NewMemoryStore()

	srcnodes, _ := src.Nodes()
	if len(srcnodes) != 2 {
		t.Fatalf("The wrong number of nodes were populated in the src repo")
	}

	inv, err := NewInventory(src)
	if err != nil {
		t.Fatalf("Unable to create inventory: %s", err)
	}

	err = inv.CopyUpdatedNodes(dst)
	if err != nil {
		t.Fatalf("Unable to copy nodes to destination: %s", err)
	}
	nodes, err := dst.Nodes()
	if err != nil {
		t.Fatalf("Unable to get nodes from destination data store: %s", err)
	}
	t.Logf("%v", nodes)

	if len(nodes) != 2 {
		t.Fatalf("The wrong number of nodes were added to the dst repo")
	}
}

func TestGetNodeByHostname(t *testing.T) {
	store, _ := NewSampleInventoryStore()
	inv, _ := NewInventory(store)
	node, err := inv.GetNodeByHostname("bar-ab01-02")
	if err != nil {
		t.Fatalf("Unable to get node: %v", err)
	}

	if node.ID() != "sample0001" {
		t.Fatalf("Wrong node returned")
	}
}
