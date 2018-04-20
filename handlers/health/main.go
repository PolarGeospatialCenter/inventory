package main

import (
	"context"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

//Health A health struct
type Health struct {
	Status int `json:"status"`
}

//Handler Documentation...
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, &Health{Status: 1})
}

func main() {
	lambda.Start(Handler)
}
