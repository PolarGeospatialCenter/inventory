package main

import (
	"context"
	"encoding/json"
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

	system := inventorytypes.NewSystem()
	system.Name = "testsystem"
	system.ShortName = "tsts"
	system.Roles = []string{"Role1", "Role2", "Role3"}
	system.Environments = map[string]*inventorytypes.Environment{
		"env": &inventorytypes.Environment{
			IPXEUrl: "http://test.com/ipxe",
		},
	}
	system.LastUpdated = time.Now()

	systemJson, err := json.Marshal(system)
	if err != nil {
		t.Errorf("unable to marshal json for system: %v", err)
	}

	modifiedSystem := *system
	modifiedSystem.Roles = []string{"FooRole"}
	modifiedSystemJson, err := json.Marshal(&modifiedSystem)
	if err != nil {
		t.Errorf("unable to marshal json for modified system: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Create test system object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       string(systemJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedStatus:     http.StatusCreated,
				ExpectedBodyObject: system,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get non-existent system",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"systemId": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get test system object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"systemId": "tsts"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: system,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Update test system object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodPut,
				PathParameters: map[string]string{"systemId": "tsts"},
				Body:           string(modifiedSystemJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedStatus:     http.StatusOK,
				ExpectedBodyObject: &modifiedSystem,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get updated test system object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"systemId": "tsts"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: &modifiedSystem,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Delete test system object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodDelete,
				PathParameters: map[string]string{"systemId": "tsts"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: "",
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get deleted test system object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"systemId": "tsts"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name:       "Attempt to get all systems",
			Request:    events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet},
			TestResult: testutils.ExpectError(http.StatusBadRequest),
		},
	}
	cases.RunTests(t, Handler)

}
