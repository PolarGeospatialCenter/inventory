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
	node.ChassisLocation = &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31}
	node.ChassisSubIndex = "a"
	node.Networks = map[string]*inventorytypes.NICInfo{
		"testnet": &inventorytypes.NICInfo{MAC: testMac},
	}
	err = inv.Update(node)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	network := inventorytypes.NewNetwork()
	network.Name = "testnetwork"
	network.MTU = 1500
	network.Metadata = make(map[string]interface{})
	_, testsubnet, _ := net.ParseCIDR("2001:db8::/64")
	network.Subnets = []*inventorytypes.Subnet{&inventorytypes.Subnet{Name: "testsubnet", Cidr: testsubnet, Gateway: net.ParseIP("2001:db8::1"), AllocationMethod: "static_inventory"}}
	err = inv.Update(network)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	type testCase struct {
		context         context.Context
		pathParameters  map[string]string
		queryParameters map[string]string
		body            string
		ExpectedBody    interface{}
		ExpectedStatus  int
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	cases := []testCase{
		testCase{handlerCtx, map[string]string{"networkId": "testnetwork", "subnetName": "testsubnet"}, map[string]string{}, `{"requestor":"testnode"}`, &IpamIpResponse{IP: net.ParseIP("2001:db8::e01c:e1fa:0:1"), Mask: 64, Gateway: net.ParseIP("2001:db8::1")}, http.StatusCreated},
		testCase{handlerCtx, map[string]string{}, map[string]string{}, "", lambdautils.ErrorResponse{Status: "Bad Request", ErrorMessage: "You must specify a network ID"}, http.StatusBadRequest},
	}

	for _, c := range cases {
		t.Logf("Testing query: %v", c.queryParameters)
		response, err := Handler(handlerCtx, events.APIGatewayProxyRequest{QueryStringParameters: c.queryParameters, HTTPMethod: http.MethodPost, PathParameters: c.pathParameters, Body: c.body})
		if err != nil {
			t.Errorf("error occurred while testing handler: %v", err)
			continue
		}

		t.Logf("Got response: %d\n%v", response.StatusCode, response.Body)
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

		case *IpamIpResponse:
			body := &IpamIpResponse{}
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
