package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/go-test/deep"
)

type TestResult struct {
	ExpectedStatus     int
	ExpectedBodyObject interface{}
}

type TestCases []TestCase

func (cases TestCases) RunTests(t *testing.T, handler func(context.Context, events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)) {
	for _, c := range cases {
		request := c.Request
		queryValues := &url.Values{}
		for k, v := range request.QueryStringParameters {
			queryValues.Add(k, v)
		}
		name := "Un-named test"
		if c.Name != "" {
			name = c.Name
		}

		response, err := handler(c.Ctx, request)
		if err != nil {
			t.Errorf("error occurred while testing handler: %v", err)
			continue
		}
		err = c.ResponseEqual(t, response)
		if err != nil {
			t.Logf("Test parameters: %s -- path params: %v -- query: %s -- body: '%s'", request.HTTPMethod, request.PathParameters, queryValues.Encode(), request.Body)
			t.Errorf("FAILED: %s -- %v", name, err)
		} else {
			t.Logf("PASSED: %s", name)
		}
	}
}

type TestCase struct {
	Name    string
	Ctx     context.Context
	Request events.APIGatewayProxyRequest
	*TestResult
}

func (c TestCase) ResponseEqual(t *testing.T, response *events.APIGatewayProxyResponse) error {
	result := c.TestResult
	status := response.StatusCode
	if status != result.ExpectedStatus {
		t.Errorf("Expected status %d, got %d", result.ExpectedStatus, status)
		return fmt.Errorf("status mismatch")
	}

	if diff := UnmarshalAndCompare(response.Body, result.ExpectedBodyObject); len(diff) > 0 {
		t.Errorf("body doesn't match expected:")
		for _, l := range diff {
			t.Errorf(l)
		}
		return fmt.Errorf("body mismatch")
	}
	return nil
}

func ExpectError(status int, msgs ...string) *TestResult {
	return &TestResult{ExpectedBodyObject: lambdautils.NewErrorResponse(status, msgs...), ExpectedStatus: status}
}

func UnmarshalAndCompare(marshaled string, obj interface{}) []string {
	body := reflect.New(reflect.TypeOf(obj))
	err := json.Unmarshal([]byte(marshaled), body.Interface())
	if err != nil {
		return []string{fmt.Sprintf("unable to unmarshal string: %v", err)}
	}
	return deep.Equal(reflect.Indirect(body).Interface(), obj)
}
