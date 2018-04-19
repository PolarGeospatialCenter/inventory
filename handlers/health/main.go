package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

//Health A health struct
type Health struct {
	Status int `json:"status"`
}

//Handler Documentation...
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	body, _ := json.Marshal(&Health{Status: 1})

	res := &events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(body),
	}

	return res, nil

}

func main() {
	lambda.Start(Handler)
}
