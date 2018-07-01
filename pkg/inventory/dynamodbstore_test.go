package inventory

import (
	"context"
	"fmt"
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
