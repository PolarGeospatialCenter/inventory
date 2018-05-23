package main

import (
	"context"
	"fmt"
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
	if len(request.QueryStringParameters) < 1 {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, fmt.Errorf("No node requested, please add query parameters"))
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	var nodeErr error
	var node *inventorytypes.InventoryNode
	if macString, ok := request.QueryStringParameters["mac"]; ok {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, err)
		}

		node, nodeErr = inv.GetInventoryNodeByMAC(mac)
		if nodeErr == nil {
			return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, node)
		}
	} else if nodeID, ok := request.QueryStringParameters["id"]; ok {
		node, nodeErr = inv.GetInventoryNodeByID(nodeID)
		if nodeErr == nil {
			return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, node)
		}
	} else {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, fmt.Errorf("invalid request, please check your parameters and try again"))
	}

	if nodeErr == inventory.ErrObjectNotFound {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotFound, map[string]string{}, nodeErr)
	}

	return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusInternalServerError, map[string]string{}, fmt.Errorf("internal server error"))
}

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return GetHandler(ctx, request)
	default:
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotImplemented, map[string]string{}, fmt.Errorf("not implemented"))
	}
}

func main() {
	lambda.Start(Handler)
}
