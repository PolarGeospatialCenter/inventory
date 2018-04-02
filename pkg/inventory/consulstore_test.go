package inventory

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"testing"
	"time"

	consul "github.com/hashicorp/consul/api"
)

var randomSrc = rand.New(rand.NewSource(time.Now().UnixNano()))

func RunConsul(ctx context.Context) (*consul.Client, error) {
	port := randomSrc.Int() % 10000
	go func() {
		port_config := fmt.Sprintf("ports { http = %d https = -1 dns = -1 serf_lan = %d serf_wan = %d server = %d}", 8500+port, 8301+port, 8302+port, 8300+port)
		cmd := exec.CommandContext(ctx, "consul", "agent", "-dev", "-hcl", port_config)
		cmd.Run()
	}()
	config := consul.DefaultConfig()
	config.Address = fmt.Sprintf("localhost:%d", 8500+port)
	client, err := consul.NewClient(config)
	return client, err
}

func TestConsul(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := RunConsul(ctx)
	if err != nil {
		t.Fatalf("Unable to start consul: %s", err)
	}

	time.Sleep(500 * time.Millisecond)
	_, err = client.Status().Leader()
	if err != nil {
		t.Fatalf("Unable to get leader from consul: %s", err)
	}
}

func TestCopyUpdatedNodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := RunConsul(ctx)
	if err != nil {
		t.Fatalf("Unable to start consul: %s", err)
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
