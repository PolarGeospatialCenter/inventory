package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

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
		// looking up an individual node
		node, err := inv.GetNodeByID(nodeId)
		switch err {
		case inventory.ErrObjectNotFound:
			return lambdautils.ErrResponse(http.StatusNotFound, err)
		case nil:
			return lambdautils.SimpleOKResponse(node)
		default:
			return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
		}
	}

	if len(request.QueryStringParameters) == 0 {
		return lambdautils.ErrStringResponse(http.StatusNotImplemented,
			"Querying all nodes is not implemented.  Please provide a filter.")
	}

	var nodeErr error
	var node *inventorytypes.Node
	if macString, ok := request.QueryStringParameters["mac"]; ok {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return lambdautils.ErrResponse(http.StatusBadRequest, err)
		}

		node, nodeErr = inv.GetNodeByMAC(mac)
		if nodeErr == nil {
			return lambdautils.SimpleOKResponse([]*inventorytypes.Node{node})
		}
	} else if nodeID, ok := request.QueryStringParameters["id"]; ok {
		node, nodeErr = inv.GetNodeByID(nodeID)
		if nodeErr == nil {
			return lambdautils.SimpleOKResponse([]*inventorytypes.Node{node})
		}
	} else {
		return lambdautils.ErrStringResponse(http.StatusBadRequest,
			"invalid request, please check your parameters and try again")
	}

	if nodeErr == inventory.ErrObjectNotFound {
		return lambdautils.ErrResponse(http.StatusNotFound, nodeErr)
	}

	return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
}

// PutHandler updates the specified node record
func PutHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	var node *inventorytypes.Node
	if nodeId, ok := request.PathParameters["nodeId"]; ok {
		var err error
		// looking up an individual node
		node, err = inv.GetNodeByID(nodeId)
		switch err {
		case inventory.ErrObjectNotFound:
			return lambdautils.ErrResponse(http.StatusNotFound, err)
		case nil:
			break
		default:
			return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
		}
	} else {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Updating all nodes not allowed.")
	}

	// parse request body.  Should be a node
	updatedNode := &inventorytypes.Node{}
	err := json.Unmarshal([]byte(request.Body), updatedNode)
	if err != nil {
		return lambdautils.ErrStringResponse(http.StatusBadRequest, "Body should contain a valid node.")
	}

	if updatedNode.InventoryID != node.InventoryID {
		return lambdautils.ErrStringResponse(http.StatusBadRequest, "Updated node inventory id must match existing.")
	}

	err = inv.Update(updatedNode)
	if err == nil {
		return lambdautils.SimpleOKResponse(updatedNode)
	}

	return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
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
		return lambdautils.ErrStringResponse(http.StatusBadRequest, "Body should contain a valid node.")
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	_, err = inv.GetNodeByID(newNode.ID())
	switch err {
	case inventory.ErrObjectNotFound:
		break
	case nil:
		return lambdautils.ErrStringResponse(http.StatusConflict, "A node with that id already exists.")
	default:
		return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
	}

	err = inv.Update(newNode)
	if err == nil {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusCreated, map[string]string{}, newNode)
	}

	return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
}

// DeleteHandler updates the specified node record
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	var node *inventorytypes.Node
	if nodeId, ok := request.PathParameters["nodeId"]; ok {
		var err error
		// looking up an individual node
		node, err = inv.GetNodeByID(nodeId)
		switch err {
		case inventory.ErrObjectNotFound:
			return lambdautils.ErrStringResponse(http.StatusNotFound, "Nodes must exist before you can delete them.")
		case nil:
			break
		default:
			return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
		}
	} else {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Deleting all nodes not allowed.")
	}

	err := inv.Delete(node)
	if err == nil {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, "")
	}

	return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
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
		return lambdautils.ErrResponse(http.StatusNotImplemented, nil)
	}
}

func main() {
	lambda.Start(Handler)
}
