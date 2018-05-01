package lambdautils

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

type ErrorResponse struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error"`
}

// NewJSONAPIGatewayProxyResponse builds a APIGatewayProxyResponse struct assuming the provided body can be marshaled into json as a map[string]interface{}
func NewJSONAPIGatewayProxyResponse(statusCode int, headers map[string]string, bodyObj interface{}) (*events.APIGatewayProxyResponse, error) {
	response := &events.APIGatewayProxyResponse{
		StatusCode:      statusCode,
		Headers:         headers,
		IsBase64Encoded: false,
	}

	if err, ok := bodyObj.(error); ok {
		bodyObj = ErrorResponse{Status: http.StatusText(statusCode), ErrorMessage: err.Error()}
	}

	bodyBytes, err := json.Marshal(bodyObj)
	if err != nil {
		return nil, err
	}
	response.Body = string(bodyBytes)

	return response, nil
}
