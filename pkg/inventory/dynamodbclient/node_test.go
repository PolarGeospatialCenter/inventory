package dynamodbclient

import (
	"context"
	"net"
	"testing"

	dynamodbtest "github.com/PolarGeospatialCenter/dockertest/pkg/dynamodb"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func TestNodeCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)

	db := dynamodb.New(session.New(dbInstance.Config()))
	inv := NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables")
	}

	err = inv.Network().Create(&types.Network{Name: "testnet"})
	if err != nil {
		t.Fatalf("unable to create network: %v", err)
	}

	mac, _ := net.ParseMAC("00-01-02-03-04-05")
	err = inv.Node().Create(
		&types.Node{
			InventoryID: "test",
			Networks: map[string]*types.NICInfo{
				"testnet": &types.NICInfo{
					MAC: mac,
				},
			},
		})

	if err != nil {
		t.Errorf("Unable to create very simple node: %v", err)
	}

	n, err := inv.Node().GetNodeByID("test")
	if err != nil {
		t.Errorf("unable to get newly created node: %v", err)
	}

	if nic, ok := n.Networks["testnet"]; !ok || nic.MAC.String() != "00:01:02:03:04:05" {
		t.Errorf("network not stored properly with node")
	}

}
