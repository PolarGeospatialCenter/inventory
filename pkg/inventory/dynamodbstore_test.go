package inventory

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"docker.io/go-docker"
	dockertypes "docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"docker.io/go-docker/api/types/network"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-test/deep"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func startDynamoDB(ctx context.Context) (*dynamodb.DynamoDB, error) {
	cli, err := docker.NewEnvClient()
	if err != nil {
		panic(err)
	}

	containerConfig := &container.Config{
		Image: "deangiberson/aws-dynamodb-local",
	}

	hostConfig := &container.HostConfig{
		PublishAllPorts: true,
	}

	networkConfig := &network.NetworkingConfig{}

	c, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, "")
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		log.Print("killing dynamodb container")
		err := cli.ContainerKill(context.Background(), c.ID, "SIGINT")
		if err != nil {
			log.Printf("Error killing dynamodb container %s: %v", c.ID, err)
		}
		cli.ContainerWait(context.Background(), c.ID, container.WaitConditionNotRunning)
		cli.ContainerRemove(context.Background(), c.ID, dockertypes.ContainerRemoveOptions{})
	}()

	err = cli.ContainerStart(ctx, c.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	log.Print(c.ID)
	containerData, err := cli.ContainerInspect(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	ports := containerData.NetworkSettings.Ports
	log.Print(ports)
	port := ports["8000/tcp"][0].HostPort
	endpoint := fmt.Sprintf("http://localhost:%s", port)
	region := "us-east-2"
	db := dynamodb.New(session.New(&aws.Config{Endpoint: &endpoint, Region: &region}))
	return db, nil
}

func loadGitStore() *GitStore {
	path, _ := os.Getwd()
	testdir := filepath.Join(path, "..", "..", "test", "data", "gitstore_2")
	cloneOpts := &git.CloneOptions{
		URL: fmt.Sprintf("file://%s", testdir),
	}
	repo, _ := git.Clone(memory.NewStorage(), nil, cloneOpts)

	return NewGitStore(repo, &git.FetchOptions{}, "master")
}

func TestDynamoDB(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := startDynamoDB(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}

	out, err := db.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		t.Errorf("Unable to list tables: %v", err)
	}
	t.Log(out.TableNames)
}

func TestDynamoDBUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := startDynamoDB(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}

	store := loadGitStore()
	store.Refresh()
	dbstore, err := NewDynamoDBStore(db, nil)
	if err != nil {
		t.Errorf("Error creating dynamo db store: %v", err)
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

	expectedTimestamp := int64(1522616442)

	if dbstore.metadata.Timestamp() != expectedTimestamp {
		t.Errorf("Last update time doesn't match expected value: %d", dbstore.metadata.Timestamp())
	}

	testStore, err := NewDynamoDBStore(db, nil)
	if err != nil {
		t.Errorf("Unable to recreate dynamodb store: %v", err)
	}
	if testStore.metadata.Timestamp() != expectedTimestamp {
		t.Errorf("Reloaded dynamodb store has incorrect timestamp: %d", testStore.metadata.Timestamp())
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

	if len(out.Items) != 3 {
		t.Errorf("Expected %d node entries, got %d", 3, len(out.Items))
	}

}
