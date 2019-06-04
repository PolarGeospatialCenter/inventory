package dynamodbclient

import (
	"context"
	"net"
	"testing"
	"time"

	dynamodbtest "github.com/PolarGeospatialCenter/dockertest/pkg/dynamodb"
	"github.com/go-test/deep"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func TestGetIPReservationInSubnetByMAC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbInstance, err := dynamodbtest.Run(ctx)
	if err != nil {
		t.Errorf("unable to start dynamodb: %v", err)
	}
	defer dbInstance.Stop(ctx)

	db := dynamodb.New(session.New(dbInstance.Config()))
	inv := NewDynamoDBStore(db, nil)

	err = inv.InitializeTables()
	if err != nil {
		t.Errorf("unable to initialize tables: %v", err)
	}

	err = inv.Network().Create(&types.Network{Name: "testnet", Subnets: types.SubnetList{
		&types.Subnet{
			Cidr: &net.IPNet{IP: net.ParseIP("10.0.0.0"), Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0)},
		},
	}})
	if err != nil {
		t.Fatalf("unable to create network: %v", err)
	}

	mac, _ := net.ParseMAC("00:01:02:03:04:05")
	r := types.NewStaticIPReservation()
	r.IP = &net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0)}
	r.MAC = mac
	zeroTime := time.Unix(0, 0)
	r.Start = &zeroTime

	err = inv.IPReservation().CreateIPReservation(r)
	if err != nil {
		t.Errorf("Unable to create IP reservation %v: %v", r, err)
	}

	mac2, _ := net.ParseMAC("00:01:02:03:04:05")
	r2 := types.NewStaticIPReservation()
	r2.IP = &net.IPNet{IP: net.ParseIP("10.0.1.1"), Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0)}
	r2.MAC = mac2
	r2.Start = &zeroTime

	err = inv.IPReservation().CreateIPReservation(r2)
	if err != nil {
		t.Errorf("Unable to create IP reservation %v: %v", r, err)
	}

	r3 := types.NewStaticIPReservation()
	r3.IP = &net.IPNet{IP: net.ParseIP("10.0.2.1"), Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0)}

	err = inv.IPReservation().CreateIPReservation(r3)
	if err != nil {
		t.Errorf("Unable to create IP reservation %v: %v", r, err)
	}

	r4 := types.NewStaticIPReservation()
	r4.IP = &net.IPNet{IP: net.ParseIP("10.0.2.2"), Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0)}

	err = inv.IPReservation().CreateIPReservation(r4)
	if err != nil {
		t.Errorf("Unable to create IP reservation %v: %v", r, err)
	}

	rResult, err := inv.IPReservation().GetExistingIPReservationInSubnet(r.IP, mac)
	if err != nil {
		t.Errorf("error getting reservation for mac in subnet: %v", err)
	}

	if diff := deep.Equal(r, rResult); len(diff) > 0 {
		t.Errorf("Reservations not equal: %v", diff)
	}

	reservations, err := inv.IPReservation().GetIPReservationsByMac(mac)
	if err != nil {
		t.Errorf("error getting reservation for mac in subnet: %v", err)
	}

	if len(reservations) != 2 {
		t.Errorf("wrong number of reservations returned")
	}

	reservations, err = inv.IPReservation().GetIPReservations(r.IP)
	if err != nil {
		t.Errorf("unable to get IP reservations in subnet: %v", err)
	}

	if len(reservations) != 1 {
		t.Errorf("wrong number of reservations returned")
	}

}
