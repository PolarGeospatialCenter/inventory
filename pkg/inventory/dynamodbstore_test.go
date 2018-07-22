package inventory

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	dynamodbtest "github.com/PolarGeospatialCenter/dockertest/pkg/dynamodb"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-test/deep"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func loadGitStore() (*GitStore, error) {
	path, _ := os.Getwd()
	testdir := filepath.Join(path, "..", "..", "test", "data", "gitstore_2")
	cloneOpts := &git.CloneOptions{
		URL: fmt.Sprintf("file://%s", testdir),
	}
	repo, err := git.Clone(memory.NewStorage(), nil, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to clone repo from %s: %v", cloneOpts.URL, err)
	}

	return NewGitStore(repo, &git.FetchOptions{}, "master"), nil
}

func TestTableMap(t *testing.T) {
	tableMap := defatultDynamoDBTables
	if table := tableMap.LookupTable(&types.Node{}); table != "inventory_nodes" {
		t.Errorf("Got wrong table name for a node: '%s'", table)
	}

	if table := tableMap.LookupTable([]*types.Node{}); table != "inventory_nodes" {
		t.Errorf("Got wrong table name for a slice of nodes: '%s'", table)
	}

	if table := tableMap.LookupTable(map[string]string{}); table != "" {
		t.Errorf("Got wrong table name for a map[string]string: '%s'", table)
	}

}

func TestDynamoDBCreateTable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)
	db := dynamodb.New(session.New(dbInstance.Config()))

	dbstore := NewDynamoDBStore(db, &DynamoDBStoreTableMap{})

	err = dbstore.createTable("test_table")
	if err != nil {
		t.Errorf("unable to create table: %v", err)
	}

	out, err := db.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		t.Errorf("unable to list tables %v", err)
	}

	// expecting 2 tables, metadata and test_table
	if len(out.TableNames) != 1 {
		t.Errorf("wrong number of tables")
	}

	if len(out.TableNames) != 1 || *out.TableNames[0] != "test_table" {
		t.Errorf("wrong table found")
	}
}

func TestDynamoDBUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)
	db := dynamodb.New(session.New(dbInstance.Config()))

	store, err := loadGitStore()
	if err != nil {
		t.Fatalf("Failed to load git store: %v", err)
	}
	store.Refresh()

	gitNodes, _ := store.GetNodes()
	numNodes := len(gitNodes)
	if numNodes != 3 {
		t.Errorf("Got %d nodes from test git store, expecting %d", numNodes, 3)
	}

	dbstore := NewDynamoDBStore(db, nil)

	err = dbstore.InitializeTables()
	if err != nil {
		t.Errorf("Error creating dynamo db store tables: %v", err)
	}

	err = dbstore.UpdateFromInventoryStore(store)
	if err != nil {
		t.Errorf("Error updating dynamodb from gitstore: %v", err)
	}

	networks, _ := store.GetNetworks()
	for _, network := range networks {
		retrieved, err := dbstore.GetNetworkByID(network.ID())
		if err != nil {
			t.Errorf("Unable to get network from dynamodb: %v", err)
		}
		if network.Metadata == nil {
			network.Metadata = retrieved.Metadata
		}
		if diff := deep.Equal(network, retrieved); len(diff) > 0 {
			for _, d := range diff {
				t.Error(d)
			}
		}

	}

	systems, _ := store.GetSystems()
	for _, system := range systems {
		retrieved, err := dbstore.GetSystemByID(system.ID())
		if err != nil {
			t.Errorf("Unable to get system from dynamodb: %v", err)
		}
		if system.Metadata == nil {
			system.Metadata = retrieved.Metadata
		}
		if diff := deep.Equal(system, retrieved); len(diff) > 0 {
			for _, d := range diff {
				t.Error(d)
			}
		}
	}

	nodes, _ := store.GetNodes()
	for _, node := range nodes {
		retrieved, err := dbstore.GetNodeByID(node.ID())
		if err != nil {
			t.Errorf("Unable to get node from dynamodb: %v", err)
		}
		if node.Metadata == nil {
			node.Metadata = retrieved.Metadata
		}
		if diff := deep.Equal(node, retrieved); len(diff) > 0 {
			for _, d := range diff {
				t.Error(d)
			}
		}

		_, err = dbstore.GetInventoryNodeByID(node.ID())
		if err != nil {
			t.Errorf("Unable to get inventory node: %v", err)
		}

		for _, nic := range node.Networks {
			retrieved, err := dbstore.GetNodeByMAC(nic.MAC)
			if err != nil {
				t.Errorf("Unable to get node from dynamodb: %v", err)
			}

			if diff := deep.Equal(node, retrieved); len(diff) > 0 {
				for _, d := range diff {
					t.Error(d)
				}
			}

			_, err = dbstore.GetInventoryNodeByMAC(nic.MAC)
			if err != nil {
				t.Errorf("Unable to get inventory node: %v", err)
			}
		}
	}

	testStore := NewDynamoDBStore(db, nil)
	if err != nil {
		t.Errorf("Unable to recreate dynamodb store: %v", err)
	}

	err = testStore.UpdateFromInventoryStore(store)
	if err != nil {
		t.Errorf("unable to update inventory from source inventory store: %v", err)
	}

	table := dbstore.tableMap.LookupTable(&types.Node{})
	out, err := dbstore.db.Scan(&dynamodb.ScanInput{TableName: &table})
	if err != nil {
		t.Errorf("Unable to scan metadata table: %v", err)
	}

	if len(out.Items) != numNodes {
		t.Errorf("Expected %d node entries, got %d", numNodes, len(out.Items))
	}

}

func TestDynamoDBDelete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)
	db := dynamodb.New(session.New(dbInstance.Config()))

	store, err := loadGitStore()
	if err != nil {
		t.Fatalf("Failed to load git store: %v", err)
	}
	store.Refresh()

	gitNodes, _ := store.GetNodes()
	numNodes := len(gitNodes)
	if numNodes != 3 {
		t.Errorf("Got %d nodes from test git store, expecting %d", numNodes, 3)
	}

	dbstore := NewDynamoDBStore(db, nil)

	err = dbstore.InitializeTables()
	if err != nil {
		t.Errorf("Error creating dynamo db store tables: %v", err)
	}

	err = dbstore.UpdateFromInventoryStore(store)
	if err != nil {
		t.Errorf("Error updating dynamodb from gitstore: %v", err)
	}

	for _, node := range gitNodes {
		err := dbstore.Delete(node)
		if err != nil {
			t.Errorf("Deletion of node %s failed: %v", node.ID(), err)
		}

		if _, err = dbstore.GetNodeByID(node.ID()); err != ErrObjectNotFound {
			t.Errorf("Found deleted node: %s (err: %v)", node.ID(), err)
		}
	}

}

func TestDynamoDBExists(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)
	db := dynamodb.New(session.New(dbInstance.Config()))

	store, err := loadGitStore()
	if err != nil {
		t.Fatalf("Failed to load git store: %v", err)
	}
	store.Refresh()

	gitNodes, _ := store.GetNodes()
	numNodes := len(gitNodes)
	if numNodes != 3 {
		t.Errorf("Got %d nodes from test git store, expecting %d", numNodes, 3)
	}

	dbstore := NewDynamoDBStore(db, nil)

	err = dbstore.InitializeTables()
	if err != nil {
		t.Errorf("Error creating dynamo db store tables: %v", err)
	}

	for _, node := range gitNodes {
		exists, err := dbstore.Exists(node)
		if exists {
			t.Errorf("Non-existent node reported as existing: %s", node.ID())
		}
		if err != nil {
			t.Errorf("Error returned from existence check: %v", err)
		}
	}

	err = dbstore.UpdateFromInventoryStore(store)
	if err != nil {
		t.Errorf("Error updating dynamodb from gitstore: %v", err)
	}

	for _, node := range gitNodes {
		exists, err := dbstore.Exists(node)
		if !exists {
			t.Errorf("Existing node reported as not existing: %s", node.ID())
		}
		if err != nil {
			t.Errorf("Error returned from existence check: %v", err)
		}
	}

}

func TestDynamoDBGetAll(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)
	db := dynamodb.New(session.New(dbInstance.Config()))

	store, err := loadGitStore()
	if err != nil {
		t.Fatalf("Failed to load git store: %v", err)
	}
	store.Refresh()

	gitNodes, _ := store.GetNodes()
	numNodes := len(gitNodes)
	if numNodes != 3 {
		t.Errorf("Got %d nodes from test git store, expecting %d", numNodes, 3)
	}

	dbstore := NewDynamoDBStore(db, nil)

	err = dbstore.InitializeTables()
	if err != nil {
		t.Errorf("Error creating dynamo db store tables: %v", err)
	}

	err = dbstore.UpdateFromInventoryStore(store)
	if err != nil {
		t.Errorf("Error updating dynamodb from gitstore: %v", err)
	}

	gitNetworks, _ := store.GetNetworks()
	dbNetworks, err := dbstore.GetNetworks()
	if err != nil {
		t.Errorf("Unable to get networks from dynamodb: %v", err)
	}

	if len(gitNetworks) != len(dbNetworks) {
		t.Errorf("Number of Networks from repo not equal to the number retrieved from dynamodb: %d != %d", len(gitNetworks), len(dbNetworks))
	}

	for _, network := range gitNetworks {
		retrieved := dbNetworks[network.ID()]
		if network.Metadata == nil {
			network.Metadata = retrieved.Metadata
		}
		if diff := deep.Equal(retrieved, network); len(diff) > 0 {
			for _, d := range diff {
				t.Error(d)
			}
		}
	}

	systems, _ := store.GetSystems()
	dbSystems, err := dbstore.GetSystems()
	if err != nil {
		t.Errorf("Unable to get systems from dynamodb: %v", err)
	}

	if len(systems) != len(dbSystems) {
		t.Errorf("Number of Systems from repo not equal to the number retrieved from dynamodb: %d != %d", len(systems), len(dbSystems))
	}

	for _, system := range systems {
		retrieved := dbSystems[system.ID()]
		if system.Metadata == nil {
			system.Metadata = retrieved.Metadata
		}
		if diff := deep.Equal(retrieved, system); len(diff) > 0 {
			for _, d := range diff {
				t.Error(d)
			}
		}
	}

	nodes, _ := store.GetNodes()
	dbNodes, err := dbstore.GetNodes()
	if err != nil {
		t.Errorf("Unable to get nodes from dynamodb: %v", err)
	}

	if len(nodes) != len(dbNodes) {
		t.Errorf("Number of Nodes from repo not equal to the number retrieved from dynamodb: %d != %d", len(nodes), len(dbNodes))
	}

	for _, node := range nodes {
		retrieved := dbNodes[node.ID()]
		if node.Metadata == nil {
			node.Metadata = retrieved.Metadata
		}
		if diff := deep.Equal(retrieved, node); len(diff) > 0 {
			for _, d := range diff {
				t.Error(d)
			}
		}
	}
}

func TestDynamoDBUpdateNodeMacs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)
	db := dynamodb.New(session.New(dbInstance.Config()))

	dbstore := NewDynamoDBStore(db, nil)

	err = dbstore.InitializeTables()
	if err != nil {
		t.Errorf("Error creating dynamo db store tables: %v", err)
	}

	mac, _ := net.ParseMAC("00:01:02:03:04:05")

	node := &types.Node{
		InventoryID: "test000",
		Networks: map[string]*types.NICInfo{
			"test1": &types.NICInfo{
				MAC: mac,
			},
		}}

	err = dbstore.Update(node)
	if err != nil {
		t.Errorf("Unable to put node: %v", err)
	}

	_, err = dbstore.GetNodeByMAC(mac)
	if err != nil {
		t.Errorf("Unable to lookup node by mac: %v", err)
	}

	node.Networks = map[string]*types.NICInfo{}
	err = dbstore.Update(node)
	if err != nil {
		t.Errorf("Unable to update node: %v", err)
	}

	_, err = dbstore.GetNodeByMAC(mac)
	if err == nil {
		t.Errorf("Node lookup by mac succeeded after removing the mac.")
	} else if err != ErrObjectNotFound {
		t.Errorf("Unexpected error looking up node by mac: %v", err)
	}

}

func TestDynamoDBDeleteNodeMacs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)
	db := dynamodb.New(session.New(dbInstance.Config()))

	dbstore := NewDynamoDBStore(db, nil)

	err = dbstore.InitializeTables()
	if err != nil {
		t.Errorf("Error creating dynamo db store tables: %v", err)
	}

	mac, _ := net.ParseMAC("00:01:02:03:04:05")

	node := &types.Node{
		InventoryID: "test000",
		Networks: map[string]*types.NICInfo{
			"test1": &types.NICInfo{
				MAC: mac,
			},
		}}

	err = dbstore.Update(node)
	if err != nil {
		t.Errorf("Unable to put node: %v", err)
	}

	_, err = dbstore.GetNodeByMAC(mac)
	if err != nil {
		t.Errorf("Unable to lookup node by mac: %v", err)
	}

	node.Networks = map[string]*types.NICInfo{}
	err = dbstore.Delete(node)
	if err != nil {
		t.Errorf("Unable to Delete node: %v", err)
	}

	_, err = dbstore.GetNodeByMAC(mac)
	if err == nil {
		t.Errorf("Node lookup by mac succeeded after removing the mac.")
	} else if err != ErrObjectNotFound {
		t.Errorf("Unexpected error looking up node by mac: %v", err)
	}

}
