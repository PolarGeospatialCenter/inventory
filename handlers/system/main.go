package main

import (
	"context"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/api/server"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	if len(request.QueryStringParameters) < 1 {
		return lambdautils.ErrBadRequest("No system requested, please add query parameters")
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	if systemID, ok := request.QueryStringParameters["id"]; ok {
		system, err := inv.GetSystemByID(systemID)
		return server.GetObjectResponse(system, err)
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
