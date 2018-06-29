package lambdautils

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

var (
	DefaultStatusMessages = map[int][]string{
		http.StatusBadRequest:          []string{"invalid request, please check your parameters and try again"},
		http.StatusNotImplemented:      []string{"not implemented"},
		http.StatusInternalServerError: []string{"An error occurred on the server.  Please report this error, an administrator will have to examine the logs to determine the cause of the error."},
		http.StatusNotFound:            []string{"not found"},
	}
)

// ErrorResponse type is what we'll use to return error objects
type ErrorResponse struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error"`
}

func NewErrorResponse(status int, msgs ...string) ErrorResponse {
	if len(msgs) == 0 {
		var ok bool
		msgs, ok = DefaultStatusMessages[status]
		if !ok {
			msgs = []string{http.StatusText(status)}
		}
	}
	return ErrorResponse{Status: http.StatusText(status), ErrorMessage: strings.Join(msgs, " ")}
}

// NewJSONAPIGatewayProxyResponse builds a APIGatewayProxyResponse struct assuming the provided body can be marshaled into json as a map[string]interface{}
func NewJSONAPIGatewayProxyResponse(statusCode int, headers map[string]string, bodyObj interface{}) (*events.APIGatewayProxyResponse, error) {
	response := &events.APIGatewayProxyResponse{
		StatusCode:      statusCode,
		Headers:         headers,
		IsBase64Encoded: false,
	}

	if err, ok := bodyObj.(error); ok {
		bodyObj = NewErrorResponse(statusCode, err.Error())
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

func ErrStringResponse(statusCode int, msgs ...string) (*events.APIGatewayProxyResponse, error) {
	return NewJSONAPIGatewayProxyResponse(statusCode, map[string]string{}, NewErrorResponse(statusCode, msgs...))
}

func ErrResponse(statusCode int, err error) (*events.APIGatewayProxyResponse, error) {
	if err == nil {
		err = errors.New(http.StatusText(statusCode))
	}
	return NewJSONAPIGatewayProxyResponse(statusCode, map[string]string{}, err)
}

func ErrNotFound(msgs ...string) (*events.APIGatewayProxyResponse, error) {
	return ErrStringResponse(http.StatusNotFound, msgs...)
}

func ErrNotImplemented(msgs ...string) (*events.APIGatewayProxyResponse, error) {
	return ErrStringResponse(http.StatusNotImplemented, msgs...)
}

func ErrBadRequest(msgs ...string) (*events.APIGatewayProxyResponse, error) {
	return ErrStringResponse(http.StatusBadRequest, msgs...)
}

func ErrInternalServerError(msgs ...string) (*events.APIGatewayProxyResponse, error) {
	return ErrStringResponse(http.StatusInternalServerError, msgs...)
}
