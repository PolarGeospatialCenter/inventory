package main

import (
	"context"
	"encoding/json"
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

	if networkID, ok := request.PathParameters["networkId"]; ok {
		network, err := inv.Network().GetNetworkByID(networkID)
		return server.GetObjectResponse(network, err)
	}

	if len(request.PathParameters) == 0 && len(request.QueryStringParameters) == 0 {
		networkMap, err := inv.Network().GetNetworks()
		networks := make([]*inventorytypes.Network, 0, len(networkMap))
		if err == nil {
			for _, n := range networkMap {
				networks = append(networks, n)
			}
		}
		return server.GetObjectResponse(networks, err)
	}

	return lambdautils.ErrBadRequest()
}

// PutHandler updates the specified network record
func PutHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	networkId, ok := request.PathParameters["networkId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Updating all networks not allowed.")
	}

	// parse request body.  Should be a network
	updatedNetwork := &inventorytypes.Network{}
	err := json.Unmarshal([]byte(request.Body), updatedNetwork)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid network.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.UpdateObject(inv.Network(), updatedNetwork, networkId)
}

// PostHandler updates the specified network record
func PostHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	if len(request.PathParameters) != 0 {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Posting not allowed here.")
	}

	// parse request body.  Should be a network
	newNetwork := &inventorytypes.Network{}
	err := json.Unmarshal([]byte(request.Body), newNetwork)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid network.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.CreateObject(inv.Network(), newNetwork)
}

// DeleteHandler updates the specified network record
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	networkId, ok := request.PathParameters["networkId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Deleting all networks not allowed.")
	}
	network := &inventorytypes.Network{Name: networkId}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.DeleteObject(inv.Network(), network)
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
