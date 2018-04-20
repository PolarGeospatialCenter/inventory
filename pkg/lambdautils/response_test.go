package lambdautils

import (
	"net/http"
	"testing"
)

func TestNewJSONAPIGatewayProxyResponseString(t *testing.T) {
	response, err := NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, "Hello World!")
	if err != nil {
		t.Fatalf("Unable to create response object from string: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Wrong status code returned: %d", response.StatusCode)
	}

	if response.Body != "Hello World!" {
		t.Errorf("Wrong body returned: %s", response.Body)
	}
}

func TestNewJSONAPIGatewayProxyResponseObject(t *testing.T) {
	response, err := NewJSONAPIGatewayProxyResponse(http.StatusOK, map[string]string{}, struct {
		Test string `json:"test"`
	}{Test: "Hello World!"})
	if err != nil {
		t.Fatalf("Unable to create response object from string: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("Wrong status code returned: %d", response.StatusCode)
	}

	if response.Body != `{"test":"Hello World!"}` {
		t.Errorf("Wrong body returned: %s", response.Body)
	}
}
