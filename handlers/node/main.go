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
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	db := dynamodb.New(session.New())
	inv := inventory.NewDynamoDBStore(db, nil)

	if len(request.QueryStringParameters) < 1 {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, "No node requested, please add query parameters")
	}

	node := &inventorytypes.Node{}
	if macString, ok := request.QueryStringParameters["mac"]; ok {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusInternalServerError, map[string]string{}, err.Error())
		}

		node, err = inv.GetNodeByMAC(mac)
		if err != nil {
			return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusInternalServerError, map[string]string{}, err.Error())
		}
	}

	if nodeID, ok := request.QueryStringParameters["nodeid"]; ok {
		var err error
		node, err = inv.GetNodeByID(nodeID)
		if err != nil {
			return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusInternalServerError, map[string]string{}, err.Error())
		}
	}

	return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, node)
}

func main() {
	lambda.Start(Handler)
}
