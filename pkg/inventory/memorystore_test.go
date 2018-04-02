package inventory

import (
	"testing"
)

func TestUpdate(t *testing.T) {
	// Test using SampleInventoryStore since that's just a populated MemoryStore
	m, err := NewSampleInventoryStore()
	if err != nil {
		t.Fatalf("Error while creating sample inventory: %v", err)
	}
	if len(m.nodes) != 2 {
		t.Fatalf("Wrong number of nodes added")
	}
}
