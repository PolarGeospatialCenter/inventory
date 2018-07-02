package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	dynamodbtest "github.com/PolarGeospatialCenter/dockertest/pkg/dynamodb"
	"github.com/PolarGeospatialCenter/inventory/pkg/api/testutils"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func TestHandler(t *testing.T) {
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
	network.LastUpdated = time.Now()

	netJson, err := json.Marshal(network)
	if err != nil {
		t.Errorf("unable to marshal network: %v", err)
	}

	updatedNetwork := *network
	updatedNetwork.MTU = 9000
	updatedNetJson, err := json.Marshal(updatedNetwork)
	if err != nil {
		t.Errorf("unable to marshal updated network: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup non-existent network",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"networkId": "testnetwork"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Create network object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       string(netJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: network,
				ExpectedStatus:     http.StatusCreated,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Attempt to create existing object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       string(netJson),
			},
			TestResult: testutils.ExpectError(http.StatusConflict, "An object with that id already exists."),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get newly created network object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"networkId": "testnetwork"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: network,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Update network object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodPut,
				PathParameters: map[string]string{"networkId": "testnetwork"},
				Body:           string(updatedNetJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: updatedNetwork,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get updated network object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"networkId": "testnetwork"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: updatedNetwork,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get list of nodes",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.Network{&updatedNetwork},
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Delete network object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodDelete,
				PathParameters: map[string]string{"networkId": "testnetwork"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: "",
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup deleted object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"networkId": "testnetwork"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Handle bad path parameters",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"badparam": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get list of nodes",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.Network{},
				ExpectedStatus:     http.StatusOK,
			},
		},
	}
	cases.RunTests(t, Handler)
}
