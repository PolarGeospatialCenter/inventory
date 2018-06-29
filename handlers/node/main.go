package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/api/server"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	if nodeId, ok := request.PathParameters["nodeId"]; ok {
		return server.GetObjectResponse(inv.GetNodeByID(nodeId))
	}

	if len(request.QueryStringParameters) == 0 {
		return lambdautils.ErrNotImplemented("Querying all nodes is not implemented.  Please provide a filter.")
	}

	if macString, ok := request.QueryStringParameters["mac"]; ok {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return lambdautils.ErrBadRequest(err.Error())
		}

		node, err := inv.GetNodeByMAC(mac)
		return server.GetObjectResponse([]*inventorytypes.Node{node}, err)
	} else if nodeID, ok := request.QueryStringParameters["id"]; ok {
		node, err := inv.GetNodeByID(nodeID)
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

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	return server.UpdateObject(inv, updatedNode, nodeId)
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

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	return server.CreateObject(inv, newNode)
}

// DeleteHandler updates the specified node record
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	nodeId, ok := request.PathParameters["nodeId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Deleting all nodes not allowed.")
	}
	node := &inventorytypes.Node{InventoryID: nodeId}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	return server.DeleteObject(inv, node)
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
