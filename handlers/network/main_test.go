package main

import (
	"context"
	"encoding/json"
	"net"
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

	network := inventorytypes.NewNetwork()
	network.Name = "testnetwork"
	network.MTU = 1500
	network.Domain = "foo"
	network.Metadata = make(map[string]interface{})
	network.Metadata["teststring"] = "test"
	network.Metadata["testint"] = float64(5)
	_, testsubnet, _ := net.ParseCIDR("10.0.0.0/24")
	network.Subnets = []*inventorytypes.Subnet{&inventorytypes.Subnet{Cidr: testsubnet}}

	err = inv.Update(network)
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
		testCase{handlerCtx, map[string]string{"id": "testnetwork"}, network, http.StatusOK},
		testCase{handlerCtx, map[string]string{"badparam": "foo"}, lambdautils.ErrorResponse{Status: "Bad Request", ErrorMessage: "invalid request, please check your parameters and try again"}, http.StatusBadRequest},
		testCase{handlerCtx, map[string]string{}, lambdautils.ErrorResponse{Status: "Bad Request", ErrorMessage: "No node requested, please add query parameters"}, http.StatusBadRequest},
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

		case *inventorytypes.Network:
			body := &inventorytypes.Network{}
			err = json.Unmarshal([]byte(response.Body), body)
			if err != nil {
				t.Errorf("Unable to unmarshal network in response")
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
