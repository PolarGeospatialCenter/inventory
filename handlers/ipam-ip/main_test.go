package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	dynamodbtest "github.com/PolarGeospatialCenter/dockertest/pkg/dynamodb"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/go-test/deep"
)

type testHandler func(ctx context.Context, t *testing.T)

func runTest(t *testing.T, h testHandler) {
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
		"testnet": &inventorytypes.NICInfo{MAC: testMac, IP: net.ParseIP("10.0.0.7")},
	}
	err = inv.Update(node)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	network := inventorytypes.NewNetwork()
	network.Name = "testnetwork"
	network.MTU = 1500
	network.Metadata = make(map[string]interface{})
	_, testsubnet, _ := net.ParseCIDR("10.0.0.0/24")
	network.Subnets = []*inventorytypes.Subnet{&inventorytypes.Subnet{Name: "testsubnet", Cidr: testsubnet, Gateway: net.ParseIP("10.0.0.1")}}
	err = inv.Update(network)
	if err != nil {
		t.Errorf("unable to create test record: %v", err)
	}

	nodeIP, nodeMask, _ := net.ParseCIDR("10.0.0.7/24")
	nodeMask.IP = nodeIP
	now := time.Now()
	end := now.Add(1 * time.Hour)
	err = inv.Update(&types.IPReservation{IP: nodeMask, MAC: testMac, Start: &now, End: &end})
	if err != nil {
		t.Errorf("Error reserving IP for test node: %v", err)
	}

	gwIp, gw, err := net.ParseCIDR("10.0.0.1/24")
	if err != nil {
		t.Errorf("Error parsing gw addr: %v", err)
	}
	gw.IP = gwIp
	err = inv.Update(&types.IPReservation{IP: gw, Start: &now})
	if err != nil {
		t.Errorf("unable to create reservation for gateway: %v", err)
	}

	gw2 := &types.IPReservation{IP: gw}
	err = inv.Get(gw2)
	if err != nil {
		t.Errorf("unable to get reservation for gateway: %v", err)
	}

	handlerCtx := lambdautils.NewAwsConfigContext(ctx, dbInstance.Config())

	h(handlerCtx, t)

}

func compareResponse(t *testing.T, response *events.APIGatewayProxyResponse, expectedResponse *events.APIGatewayProxyResponse) {
	if diff := deep.Equal(response, expectedResponse); len(diff) > 0 {
		t.Errorf("response doesn't match expected:")
		for _, l := range diff {
			t.Errorf(l)
		}
	}
}

func TestCreateReservationUnknownHost(t *testing.T) {
	runTest(t, func(handlerCtx context.Context, t *testing.T) {
		// Post to ip endpoint with MAC, network/subnet and hostname, no IP.  Should return an IP from the subnet.
		response, err := Handler(handlerCtx, events.APIGatewayProxyRequest{
			HTTPMethod: http.MethodPost,
			Body: `
			{
				"mac": "02:03:04:05:06:07",
				"name": "foo-host",
				"subnet": "10.0.0.0",
				"ttl": "1h"
			}`,
		})
		if err != nil {
			t.Fatalf("Unexpected error creating reservation for unknown host: %v", err)
		}
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("Expected created status, got: %d", response.StatusCode)
		}

		reservation := &types.IpamIpResponse{}
		err = json.Unmarshal([]byte(response.Body), reservation)
		if err != nil {
			t.Fatalf("Unable to parse response: %v", err)
		}

		_, expectedNet, _ := net.ParseCIDR("10.0.0.0/24")
		reservedIP, _, _ := net.ParseCIDR(reservation.IP)
		if !expectedNet.Contains(reservedIP) {
			t.Errorf("Reserved IP in wrong subnet: %s", reservation.IP)
		}

		t.Log(reservation)
		if reservation.Start == nil {
			t.Errorf("Got nil start time")
		}

		if reservation.End == nil {
			t.Errorf("Got nil end time")
		}

	})
}

func TestCreateReservationKnownHost(t *testing.T) {
	runTest(t, func(handlerCtx context.Context, t *testing.T) {
		// Post to ip endpoint with MAC, network/subnet and hostname, no IP.  Sound return a conflict.
		response, err := Handler(handlerCtx, events.APIGatewayProxyRequest{
			HTTPMethod: http.MethodPost,
			Body: `
			{
				"mac": "00:01:02:03:04:05",
				"subnet": "10.0.0.0"
			}`,
		})
		if err != nil {
			t.Fatalf("Unexpected error creating reservation for unknown host: %v", err)
		}
		if response.StatusCode != http.StatusConflict {
			t.Fatalf("Expected conflict status, got: %d", response.StatusCode)
		}
	})
}

func TestGetReservationKnownHost(t *testing.T) {
	runTest(t, func(handlerCtx context.Context, t *testing.T) {
		// Post to ip endpoint with MAC, network/subnet and hostname, no IP.  Sound return a conflict.
		response, err := Handler(handlerCtx, events.APIGatewayProxyRequest{
			HTTPMethod:     http.MethodGet,
			PathParameters: map[string]string{"ipAddress": "10.0.0.7"},
		})
		if err != nil {
			t.Fatalf("Unexpected error creating reservation for unknown host: %v", err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("Expected ok status, got: %d", response.StatusCode)
		}
		t.Log(response.Body)

		r := &types.IpamIpResponse{}
		err = json.Unmarshal([]byte(response.Body), r)
		if err != nil {
			t.Errorf("Unable to unmarshal response: %v", err)
		}
		if r.Gateway != "10.0.0.1" {
			t.Errorf("Gateway value doesn't match expected %v", r.Gateway)
		}
		if r.Start == nil {
			t.Errorf("Got nil start time")
		}

		if r.End == nil {
			t.Errorf("Got nil end time")
		}
	})
}
func TestCreateReservationStaticReservation(t *testing.T) {
	runTest(t, func(handlerCtx context.Context, t *testing.T) {
		// Post to ip endpoint with MAC, and IP staticly reserved in subnet.  Should return Conflict.
		response, err := Handler(handlerCtx, events.APIGatewayProxyRequest{
			HTTPMethod:     http.MethodPost,
			PathParameters: map[string]string{"ipAddress": "10.0.0.2"},
			Body: `
			{
				"mac": "01:02:03:04:05:06",
				"name": "test-entry"
			}`,
		})
		if err != nil {
			t.Fatalf("Unexpected error creating reservation for unknown host: %v", err)
		}
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("Expected created status, got: %d", response.StatusCode)
		}

		reservation := &types.IpamIpResponse{}
		err = json.Unmarshal([]byte(response.Body), reservation)
		if err != nil {
			t.Log(response.Body)
			t.Fatalf("Unable to parse response: %v", err)
		}

		if reservation.IP != "10.0.0.2/24" {
			t.Errorf("Wrong IP reserved: %s", reservation.IP)
		}
	})
}

// func TestCreateReservationNodeConflict(t *testing.T) {
// 	runTest(t, func(handlerCtx context.Context, t *testing.T) {
// 		// Post to ip endpoint with MAC, and IP already reserved for another host.  Should return Conflict.
// 		t.Errorf("not implemented")
// 	})
// }

func TestCreateReservationConflict(t *testing.T) {
	runTest(t, func(handlerCtx context.Context, t *testing.T) {
		// Post to ip endpoint with MAC, and IP already dynamically reserved for another host.  Should return Conflict.
		response, err := Handler(handlerCtx, events.APIGatewayProxyRequest{
			HTTPMethod:     http.MethodPost,
			PathParameters: map[string]string{"ipAddress": "10.0.0.1"},
			Body: `
			{
				"mac": "01:02:03:04:05:06",
				"name": "test-entry"
			}`,
		})
		if err != nil {
			t.Fatalf("Unexpected error creating reservation for unknown host: %v", err)
		}
		if response.StatusCode != http.StatusConflict {
			t.Fatalf("Expected conflict status, got: %d", response.StatusCode)
		}
	})
}

// func TestCreateReservationExpiredConflict(t *testing.T) {
// 	runTest(t, func(handlerCtx context.Context, t *testing.T) {
// 		// Post to ip endpoint with MAC, and IP already reserved for another host - but has expired.  Should return success.
// 		t.Errorf("not implemented")
// 	})
// }

// func TestGetReservation(t *testing.T) {
// 	runTest(t, func(handlerCtx context.Context, t *testing.T) {
// 		// Create a reservation, then retrieve it.
// 		t.Errorf("not implemented")
// 	})
// }
