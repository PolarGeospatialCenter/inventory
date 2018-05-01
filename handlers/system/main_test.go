package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	dynamodbtest "github.com/PolarGeospatialCenter/dockertest/pkg/dynamodb"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-test/deep"
)

func TestGetHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)

	db := dynamodb.New(session.New(dbInstance.Config()))
	inv := inventory.NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables")
	}

	system := inventorytypes.NewSystem()
	system.Name = "testsystem"
	system.ShortName = "tsts"
	system.Roles = []string{"Role1", "Role2", "Role3"}
	system.Environments = map[string]*inventorytypes.Environment{
		"env": &inventorytypes.Environment{
			IPXEUrl: "http://test.com/ipxe",
		},
	}

	err = inv.Update(system)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	type testCase struct {
		context         context.Context
		queryParameters map[string]string
		ExpectedBody    interface{}
		ExpectedStatus  int
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := []testCase{
		testCase{handlerCtx, map[string]string{"id": "foo"}, lambdautils.ErrorResponse{Status: "Not Found", ErrorMessage: "Object not found"}, http.StatusNotFound},
		testCase{handlerCtx, map[string]string{"id": "tsts"}, system, http.StatusOK},
		testCase{handlerCtx, map[string]string{"badparam": "foo"}, lambdautils.ErrorResponse{Status: "Bad Request", ErrorMessage: "invalid request, please check your parameters and try again"}, http.StatusBadRequest},
		testCase{handlerCtx, map[string]string{}, lambdautils.ErrorResponse{Status: "Bad Request", ErrorMessage: "No system requested, please add query parameters"}, http.StatusBadRequest},
	}

	for _, c := range cases {
		t.Logf("Testing query: %v", c.queryParameters)
		response, err := Handler(handlerCtx, events.APIGatewayProxyRequest{QueryStringParameters: c.queryParameters, HTTPMethod: http.MethodGet})
		if err != nil {
			t.Errorf("error occurred while testing handler: %v", err)
			continue
		}

		status := response.StatusCode
		if status != c.ExpectedStatus {
			t.Errorf("Expected status %d, got %d", c.ExpectedStatus, status)
		}

		switch c.ExpectedBody.(type) {
		case lambdautils.ErrorResponse:
			body := lambdautils.ErrorResponse{}
			err = json.Unmarshal([]byte(response.Body), &body)
			if err != nil {
				t.Errorf("Unable to unmarshal error in response: %v", err)
			}

			if diff := deep.Equal(body, c.ExpectedBody); len(diff) > 0 {
				t.Errorf("body doesn't match expected:")
				for _, l := range diff {
					t.Errorf(l)
				}
			}

		case *inventorytypes.System:
			body := &inventorytypes.System{}
			err = json.Unmarshal([]byte(response.Body), body)
			if err != nil {
				t.Errorf("Unable to unmarshal node in response")
			}

			if diff := deep.Equal(body, c.ExpectedBody); len(diff) > 0 {
				t.Errorf("body doesn't match expected:")
				for _, l := range diff {
					t.Errorf(l)
				}
			}
		default:
			t.Errorf("You've specified a return type that isn't implemented for testing.")
		}
	}

}
