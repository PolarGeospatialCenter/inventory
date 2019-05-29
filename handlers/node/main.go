package main

import (
	"context"
	"encoding/json"
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
		return server.GetObjectResponse(inv.Node().GetNodeByID(nodeId))
	}

	if len(request.QueryStringParameters) == 0 {
		nodeMap, err := inv.Node().GetNodes()
		nodes := make([]*inventorytypes.Node, 0, len(nodeMap))
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

		node, err := inv.Node().GetNodeByMAC(mac)
		return server.GetObjectResponse([]*inventorytypes.Node{node}, err)
	} else if nodeID, ok := request.QueryStringParameters["id"]; ok {
		node, err := inv.Node().GetNodeByID(nodeID)
		return server.GetObjectResponse([]*inventorytypes.Node{node}, err)
	}

	return lambdautils.ErrBadRequest()
}

// PutHandler updates the specified node record
func PutHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	nodeId, ok := request.PathParameters["nodeId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Updating all nodes not allowed.")
	}

	// parse request body.  Should be a node
	updatedNode := &inventorytypes.Node{}
	err := json.Unmarshal([]byte(request.Body), updatedNode)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid node.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.UpdateObject(inv.Node(), updatedNode, nodeId)
}

// PostHandler updates the specified node record
func PostHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	if len(request.PathParameters) != 0 {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Posting not allowed here.")
	}

	// parse request body.  Should be a node
	newNode := &inventorytypes.Node{}
	err := json.Unmarshal([]byte(request.Body), newNode)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid node.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.CreateObject(inv.Node(), newNode)
}

// DeleteHandler updates the specified node record
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	nodeId, ok := request.PathParameters["nodeId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Deleting all nodes not allowed.")
	}
	node := &inventorytypes.Node{InventoryID: nodeId}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.DeleteObject(inv.Node(), node)
}

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return GetHandler(ctx, request)
	case http.MethodPut:
		return PutHandler(ctx, request)
	case http.MethodPost:
		return PostHandler(ctx, request)
	case http.MethodDelete:
		return DeleteHandler(ctx, request)
	default:
		return lambdautils.ErrNotImplemented()
	}
}

func main() {
	lambda.Start(Handler)
}
