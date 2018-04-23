package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHealthHandler(t *testing.T) {
	req := events.APIGatewayProxyRequest{}
	response, err := Handler(context.Background(), req)
	if err != nil {
		t.Errorf("test of handler failed")
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("wrong status code returned (not 200): %d", response.StatusCode)
	}

	h := &Health{}
	err = json.Unmarshal([]byte(response.Body), h)
	if err != nil {
		t.Errorf("unable to unmarshal json body: %v", err)
	}

	if h.Status != 1 {
		t.Errorf("wrong status returned from health check: %d", h.Status)
	}
}
