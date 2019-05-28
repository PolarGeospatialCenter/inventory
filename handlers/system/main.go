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

	if systemID, ok := request.PathParameters["systemId"]; ok {
		system, err := inv.GetSystemByID(systemID)
		return server.GetObjectResponse(system, err)
	}

	if len(request.PathParameters) == 0 && len(request.QueryStringParameters) == 0 {
		systemMap, err := inv.GetSystems()
		systems := make([]*inventorytypes.System, 0, len(systemMap))
		if err == nil {
			for _, n := range systemMap {
				systems = append(systems, n)
			}
		}
		return server.GetObjectResponse(systems, err)
	}

	return lambdautils.ErrBadRequest()
}

// PutHandler updates the specified system record
func PutHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	systemId, ok := request.PathParameters["systemId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Updating all systems not allowed.")
	}

	// parse request body.  Should be a system
	updatedSystem := &inventorytypes.System{}
	err := json.Unmarshal([]byte(request.Body), updatedSystem)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid system.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.UpdateObject(inv, updatedSystem, systemId)
}

// PostHandler updates the specified system record
func PostHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	if len(request.PathParameters) != 0 {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Posting not allowed here.")
	}

	// parse request body.  Should be a system
	newSystem := &inventorytypes.System{}
	err := json.Unmarshal([]byte(request.Body), newSystem)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid system.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.CreateObject(inv, newSystem)
}

// DeleteHandler updates the specified system record
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	systemId, ok := request.PathParameters["systemId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Deleting all systems not allowed.")
	}
	system := &inventorytypes.System{Name: systemId}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.DeleteObject(inv, system)
}

// Handler handles requests for systems
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
