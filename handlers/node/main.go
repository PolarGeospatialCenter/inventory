package main

import (
	"context"
	"net"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*inventorytypes.InventoryNode, error) {
	db := dynamodb.New(session.New())
	inv := inventory.NewDynamoDBStore(db, nil)

	node := &inventorytypes.InventoryNode{}
	if macString, ok := request.QueryStringParameters["mac"]; ok {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return nil, err
		}

		node, err = inv.GetInventoryNodeByMAC(mac)
		if err != nil {
			return nil, err
		}
	}

	if nodeID, ok := request.QueryStringParameters["nodeid"]; ok {
		var err error
		node, err = inv.GetInventoryNodeByID(nodeID)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

func main() {
	lambda.Start(Handler)
}
