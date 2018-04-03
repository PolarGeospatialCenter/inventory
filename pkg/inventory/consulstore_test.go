package inventory

import (
	"context"
	"testing"

	consultest "github.com/PolarGeospatialCenter/dockertest/pkg/consul"
	consul "github.com/hashicorp/consul/api"
)

func TestCopyUpdatedNodes(t *testing.T) {
	ctx := context.Background()
	instance, err := consultest.Run(ctx)
	if err != nil {
		t.Fatalf("unable to start consul: %v", err)
	}
	defer instance.Stop(ctx)
	client, err := consul.NewClient(instance.Config())
	if err != nil {
		t.Fatalf("Unable to create consul client: %v", err)
	}

	cStore, err := NewConsulStore(client, "testbase")
	if err != nil {
		t.Fatalf("Unable to create consul data store: %s", err)
	}

	initNodes, _ := cStore.Nodes()
	if len(initNodes) != 0 {
		t.Fatalf("Consul inventory not empty.")
	}

	sample, _ := NewSampleInventoryStore()
	inv, _ := NewInventory(sample)
	err = inv.CopyUpdatedNodes(cStore)
	if err != nil {
		t.Fatalf("Error copying sample nodes to consul: %v", err)
	}

	sampleNodes, _ := sample.Nodes()
	// Check to see if MTU marshaled properly
	nodes, err := cStore.Nodes()
	if err != nil {
		t.Fatalf("Unable to read nodes from consul store")
	}

	node, ok := nodes["sample0001"]
	if !ok {
		t.Fatalf("sample0001 doesn't exist in datastore")
	}

	if node.Networks["provisioning"].Network.MTU != sampleNodes["sample0001"].Networks["provisioning"].Network.MTU {
		key, _ := cStore.InventoryObjectKey(sampleNodes["sample0001"])
		pair, meta, _ := client.KV().Get(key, &consul.QueryOptions{})
		t.Logf("Consul data: %d %v", meta.LastIndex, string(pair.Value))
		marshaled, _ := Marshal(sampleNodes["sample0001"])
		t.Logf("Marshal data: %v", string(marshaled))
		t.Fatalf("MTU not marshaled properly: %v", node.Networks["provisioning"].Network.MTU)
	}

	// Intentionally corrupt a node
	key, _ := cStore.InventoryObjectKey(sampleNodes["sample0001"])
	pair := &consul.KVPair{Key: key, Value: []byte("{Bad Data}")}
	_, err = client.KV().Put(pair, &consul.WriteOptions{})
	if err != nil {
		t.Fatalf("Unable to corrupt node in consul for testing: %v", err)
	}

	nodes, err = cStore.Nodes()
	if err != nil {
		t.Fatalf("Unable to read nodes from consul store after corrupting a node")
	}

	_, ok = nodes["sample0001"]
	if ok {
		t.Fatalf("Node still returned after being corrupted in data store")
	}

	err = inv.CopyUpdatedNodes(cStore)
	if err != nil {
		t.Fatalf("Error updating nodes in consul: %v", err)
	}

	nodes, err = cStore.Nodes()
	if err != nil {
		t.Fatalf("Unable to read nodes from consul store")
	}

	node, ok = nodes["sample0001"]
	if !ok {
		t.Fatalf("sample0001 couldn't be loaded after re-copying nodes")
	}

}
