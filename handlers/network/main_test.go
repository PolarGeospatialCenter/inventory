package main

import (
	"context"
	"net"
	"net/http"
	"testing"

	dynamodbtest "github.com/PolarGeospatialCenter/dockertest/pkg/dynamodb"
	"github.com/PolarGeospatialCenter/inventory/pkg/api/testutils"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"networkId": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
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
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"badparam": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest),
		},
	}
	cases.RunTests(t, Handler)
}
