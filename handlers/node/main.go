package main

import (
	"context"
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

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return GetHandler(ctx, request)
	default:
		return lambdautils.ErrResponse(http.StatusNotImplemented, nil)
	}
}

func main() {
	lambda.Start(Handler)
}
