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
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/dynamodbclient"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
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
	node.Tags = inventorytypes.Tags{}
	node.Metadata = inventorytypes.Metadata{}
	node.LastUpdated = time.Now()
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
	inv := dynamodbclient.NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables")
	}

	err = inv.Network().Create(&types.Network{Name: "testnet"})
	if err != nil {
		t.Fatalf("unable to create network: %v", err)
	}

	node := testNode()
	err = inv.Node().Create(node)
	if err != nil {
		t.Fatalf("unable to create test record: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get non-existent object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				PathParameters:        map[string]string{},
				QueryStringParameters: map[string]string{"id": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get node via query for id",
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
			Name: "Get node via nodeId path parameter",
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
			Name: "Lookup non-existent node by MAC",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": "01:02:03:04:05:06"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup node by MAC",
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
			Name: "Lookup node by MAC with extraneous query parameters",
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
			Name: "Test MAC address query input validation",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest, "address foo: invalid MAC address"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Bad query parameter",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"badparam": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name:    "Get all nodes",
			Request: events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.Node{node},
				ExpectedStatus:     http.StatusOK,
			},
		},
	}
	cases.RunTests(t, Handler)
}

func TestGetHandlerNullEntries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)

	db := dynamodb.New(session.New(dbInstance.Config()))
	inv := dynamodbclient.NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables")
	}

	err = inv.Network().Create(&types.Network{Name: "testnet"})
	if err != nil {
		t.Fatalf("unable to create network: %v", err)
	}

	node := &inventorytypes.Node{InventoryID: "test-0034"}
	err = inv.Node().Create(node)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	// expect maps/lists to be returned as empty rather than nil
	node.Networks = inventorytypes.NICInfoMap{}
	node.Metadata = inventorytypes.Metadata{}
	node.Tags = []string{}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get node via nodeId path parameter",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"nodeId": "test-0034"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: node,
				ExpectedStatus:     http.StatusOK,
			},
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
	inv := dynamodbclient.NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables")
	}

	testNet := &types.Network{Name: "testnet"}
	err = inv.Network().Update(testNet)
	if err != nil {
		t.Fatalf("unable to create test network: %v", err)
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
			Name: "Create new node",
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
			Name: "Update node object",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodPut,
				PathParameters: map[string]string{"nodeId": "testnode"},
				Body:           string(updatedNodeJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: updatedNode,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get updated node",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"nodeId": "testnode"},
				Body:           string(updatedNodeJson),
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: &updatedNode,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Attempt updating all nodes",
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
	inv := dynamodbclient.NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables")
	}

	testNet := &types.Network{
		Name: "testnet",
		Subnets: types.SubnetList{
			&types.Subnet{
				Cidr: &net.IPNet{
					IP:   net.ParseIP("10.0.0.1"),
					Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0x00),
				},
				AllocationMethod: "static_inventory",
			},
		},
	}
	err = inv.Network().Update(testNet)
	if err != nil {
		t.Fatalf("unable to create test network: %v", err)
	}

	node := testNode()
	node.InventoryID = "testnode-002"

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	nodeJson, err := json.Marshal(node)
	if err != nil {
		t.Errorf("Unable to marshal node json: %v", err)
	}

	node.Networks["testnet"].IP = net.ParseIP("10.0.0.2")

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Create new node",
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
			Name: "Attempt to create new node again",
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
	inv := dynamodbclient.NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables")
	}

	err = inv.Network().Create(&types.Network{Name: "testnet"})
	if err != nil {
		t.Fatalf("unable to create network: %v", err)
	}

	node := testNode()

	err = inv.Node().Create(node)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Delete test node",
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
			Name: "Attempt to delete node again",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodDelete,
				PathParameters: map[string]string{"nodeId": "testnode"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Objects must exist before you can delete them."),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Attempt to delete all nodes",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodDelete,
			},
			TestResult: testutils.ExpectError(http.StatusMethodNotAllowed, "Deleting all nodes not allowed."),
		},
	}
	cases.RunTests(t, Handler)
}
