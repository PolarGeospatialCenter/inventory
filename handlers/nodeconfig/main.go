package main

import (
	"context"
	"net"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/api/server"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	inv := server.ConnectToInventoryFromContext(ctx)

	if nodeId, ok := request.PathParameters["nodeId"]; ok {
		// looking up an individual node
		node, err := inv.GetInventoryNodeByID(nodeId)
		return server.GetObjectResponse(node, err)
	}

	if len(request.QueryStringParameters) == 0 {
		nodeMap, err := inv.GetInventoryNodes()
		nodes := make([]*inventorytypes.InventoryNode, 0, len(nodeMap))
		if err == nil {
			for _, n := range nodeMap {
				nodes = append(nodes, n)
			}
		}
		return server.GetObjectResponse(nodes, err)
	}

	if macString, ok := request.QueryStringParameters["mac"]; ok {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return lambdautils.ErrBadRequest(err.Error())
		}

		node, err := inv.GetInventoryNodeByMAC(mac)
		return server.GetObjectResponse([]*inventorytypes.InventoryNode{node}, err)
	} else if nodeID, ok := request.QueryStringParameters["id"]; ok {
		node, err := inv.GetInventoryNodeByID(nodeID)
		return server.GetObjectResponse([]*inventorytypes.InventoryNode{node}, err)
	}

	return lambdautils.ErrBadRequest()
}

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return GetHandler(ctx, request)
	default:
		return lambdautils.ErrNotImplemented()
	}
}

func main() {
	lambda.Start(Handler)
}
