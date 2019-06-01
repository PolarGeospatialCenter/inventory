package main

import (
	"context"
	"net"
	"net/http"
	"testing"

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

	node := inventorytypes.NewNode()
	node.InventoryID = "testnode"
	testMac, _ := net.ParseMAC("00:01:02:03:04:05")
	node.System = "tsts"
	node.Environment = "env"
	node.Role = "Role1"
	node.Tags = inventorytypes.Tags{}
	node.Metadata = inventorytypes.Metadata{}
	node.Networks = types.NICInfoMap{
		"testnetwork": &inventorytypes.NetworkInterface{NICs: []net.HardwareAddr{testMac}, Metadata: types.Metadata{}},
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

	err = inv.Network().Create(network)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	system := inventorytypes.NewSystem()
	system.Name = "testsystem"
	system.ShortName = "tsts"
	system.Roles = []string{"Role1", "Role2", "Role3"}
	system.Environments = map[string]*inventorytypes.Environment{
		"env": &inventorytypes.Environment{
			IPXEUrl: "http://test.com/ipxe",
			Networks: map[string]string{
				"testnet": "testnetwork",
			},
		},
	}
	system.Metadata = inventorytypes.Metadata{}

	err = inv.System().Create(system)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	err = inv.Node().Create(node)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())
	inv.IPReservation().CreateIPReservation(&types.IPReservation{
		IP:  &net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0)},
		MAC: testMac,
	})

	inventoryNode, err := inventorytypes.NewInventoryNode(node, inventorytypes.NetworkMap{"testnetwork": network}, inventorytypes.SystemMap{"tsts": system}, inv.IPReservation())
	if err != nil {
		t.Errorf("unable to build inventory node: %v", err)
	}

	cases := testutils.TestCases{
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup non-existent node",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"id": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup test node using path parameter",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:     http.MethodGet,
				PathParameters: map[string]string{"nodeId": "testnode"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: inventoryNode,
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup test node by id query",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"id": "testnode"},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.InventoryNode{inventoryNode},
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup non-existent node by mac",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": "01:02:03:04:05:06"},
			},
			TestResult: testutils.ExpectError(http.StatusNotFound, "Object not found"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Lookup test node by mac",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": testMac.String()},
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.InventoryNode{inventoryNode},
				ExpectedStatus:     http.StatusOK,
			},
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Test mac input validation",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod:            http.MethodGet,
				QueryStringParameters: map[string]string{"mac": "foo"},
			},
			TestResult: testutils.ExpectError(http.StatusBadRequest, "address foo: invalid MAC address"),
		},
		testutils.TestCase{Ctx: handlerCtx,
			Name: "Get all nodes",
			Request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
			},
			TestResult: &testutils.TestResult{
				ExpectedBodyObject: []*inventorytypes.InventoryNode{inventoryNode},
				ExpectedStatus:     http.StatusOK,
			},
		},
	}
	cases.RunTests(t, Handler)
}
