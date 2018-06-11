package lambdautils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// ErrorResponse type is what we'll use to return error objects
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

func SimpleOKResponse(result interface{}) (*events.APIGatewayProxyResponse, error) {
	return NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, result)
}

func ErrStringResponse(statusCode int, msg string) (*events.APIGatewayProxyResponse, error) {
	return NewJSONAPIGatewayProxyResponse(statusCode, map[string]string{}, fmt.Errorf(msg))
}

func ErrResponse(statusCode int, err error) (*events.APIGatewayProxyResponse, error) {
	if err == nil {
		err = errors.New(http.StatusText(statusCode))
	}
	return NewJSONAPIGatewayProxyResponse(statusCode, map[string]string{}, err)
}
