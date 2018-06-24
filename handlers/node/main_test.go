package main

import (
	"context"
	"encoding/json"
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

func testNode() *inventorytypes.Node {
	node := inventorytypes.NewNode()
	node.InventoryID = "testnode"
	testMac, _ := net.ParseMAC("00:01:02:03:04:05")
	node.Networks = map[string]*inventorytypes.NICInfo{
		"testnet": &inventorytypes.NICInfo{MAC: testMac},
	}
	return node
}

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

	node := testNode()
	err = inv.Update(node)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				PathParameters:        map[string]string{},
				QueryStringParameters: map[string]string{"id": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"id": "testnode"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.Node{node},
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"nodeId": "testnode"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: node,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": "01:02:03:04:05:06"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": node.Networks["testnet"].MAC.String()},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.Node{node},
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": node.Networks["testnet"].MAC.String(), "badparam": "baz"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.Node{node},
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest, "address foo: invalid MAC address"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"badparam": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request:    events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet},
			TestResult: testutils.ExpectError(http.StatusNotImplemented, "Querying all nodes is not implemented.  Please provide a filter."),
		},
	}
	cases.RunTests(t, Handler)
}

func TestPutHandler(t *testing.T) {
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

	node := testNode()

	nodeJson, err := json.Marshal(node)
	if err != nil {
		t.Errorf("Unable to marshal node json: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	updatedMac, _ := net.ParseMAC("01:02:03:04:05:06")
	updatedNode := *node
	updatedNode.Networks = map[string]*inventorytypes.NICInfo{
		"testnet": &inventorytypes.NICInfo{MAC: updatedMac},
	}
	updatedNodeJson, err := json.Marshal(updatedNode)
	if err != nil {
		t.Errorf("Unable to marshal updated node json: %v", err)
	}

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       string(nodeJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: node,
				ExpectedStatus:     http.StatusCreated,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodPut,
				PathParameters: map[string]string{"nodeId": "testnode"},
				Body:           string(updatedNodeJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: &updatedNode,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPut,
			},
			TestResult: testutils.ExpectError(http.StatusMethodNotAllowed, "Updating all nodes not allowed."),
		},
	}
	cases.RunTests(t, Handler)
}

func TestPostHandler(t *testing.T) {
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

	node := testNode()

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	nodeJson, err := json.Marshal(node)
	if err != nil {
		t.Errorf("Unable to marshal node json: %v", err)
	}

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       string(nodeJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: node,
				ExpectedStatus:     http.StatusCreated,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Body:       string(nodeJson),
			},
			TestResult: testutils.ExpectError(http.StatusConflict, "An object with that id already exists."),
		},
	}
	cases.RunTests(t, Handler)
}

func TestDeleteHandler(t *testing.T) {
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

	node := testNode()

	err = inv.Update(node)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodDelete,
				PathParameters: map[string]string{"nodeId": "testnode"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: "",
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodDelete,
				PathParameters: map[string]string{"nodeId": "testnode"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Objects must exist before you can delete them."),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodDelete,
			},
			TestResult: testutils.ExpectError(http.StatusMethodNotAllowed, "Deleting all nodes not allowed."),
		},
	}
	cases.RunTests(t, Handler)
}
