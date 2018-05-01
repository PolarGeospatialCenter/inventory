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

	node := inventorytypes.NewNode()
	node.InventoryID = "testnode"
	testMac, _ := net.ParseMAC("00:01:02:03:04:05")
	node.Networks = map[string]*inventorytypes.NICInfo{
		"testnet": &inventorytypes.NICInfo{MAC: testMac},
	}

	err = inv.Update(node)
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
		testCase{handlerCtx, map[string]string{"id": "foo"}, "Object not found", http.StatusNotFound},
		testCase{handlerCtx, map[string]string{"id": "testnode"}, node, http.StatusOK},
		testCase{handlerCtx, map[string]string{"mac": "01:02:03:04:05:06"}, "Object not found", http.StatusNotFound},
		testCase{handlerCtx, map[string]string{"mac": testMac.String()}, node, http.StatusOK},
		testCase{handlerCtx, map[string]string{"mac": testMac.String(), "badparam": "baz"}, node, http.StatusOK},
		testCase{handlerCtx, map[string]string{"mac": "foo"}, "address foo: invalid MAC address", http.StatusBadRequest},
		testCase{handlerCtx, map[string]string{"badparam": "foo"}, "invalid request, please check your parameters and try again", http.StatusBadRequest},
		testCase{handlerCtx, map[string]string{}, "No node requested, please add query parameters", http.StatusBadRequest},
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
		case string:
			body := ""
			err = json.Unmarshal([]byte(response.Body), &body)
			if err != nil {
				t.Errorf("Unable to unmarshal string in response")
			}

			if body != c.ExpectedBody.(string) {
				t.Errorf("body doesn't match: expected '%s', got '%s'", c.ExpectedBody.(string), body)
			}
		case *inventorytypes.Node:
			body := &inventorytypes.Node{}
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
		}
	}

}
