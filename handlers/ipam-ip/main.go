package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	ip, ok := request.PathParameters["ipAddress"]
	if !ok {
		return lambdautils.ErrBadRequest("You must specify an IP address")
	}

	ipAddress := net.ParseIP(ip)
	if ipAddress == nil {
		return lambdautils.ErrBadRequest("Bad IP address")
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	// lookup network and subnet
	subnet, err := lookupSubnetForIP(inv, ipAddress)
	if err != nil {
		log.Printf("unable to lookup subnet for IP %s: %v", ipAddress, err)
		return lambdautils.ErrInternalServerError("consult logs for details")
	}

	reservation, err := inv.GetIPReservation(&net.IPNet{IP: ipAddress, Mask: subnet.Cidr.Mask})
	if err != nil {
		return lambdautils.ErrNotFound("No reservation found for that IP")
	}

	reservation.SetSubnetInformation(subnet)
	return lambdautils.SimpleOKResponse(reservation)
}

func lookupSubnetForIP(inv *inventory.DynamoDBStore, ip net.IP) (*types.Subnet, error) {
	networks, err := inv.GetNetworks()
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		for _, subnet := range network.Subnets {
			if subnet.Cidr.Contains(ip) {
				return subnet, nil
			}
		}
	}
	return nil, nil

}

// PutHandler handles POST method requests from the API gateway
func PutHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	ipReservation := &types.IPReservation{}
	err := json.Unmarshal([]byte(request.Body), ipReservation)
	if err != nil {
		log.Printf("Unable to parse request: %v", err)
		return lambdautils.ErrBadRequest("Unable to parse request")
	}

	ipAddress := request.PathParameters["ipAddress"]
	ip := net.ParseIP(ipAddress)
	if ip == nil && ipAddress != "" {
		return lambdautils.ErrBadRequest("invalid IP address")
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	existingReservation := &types.IPReservation{}
	existingReservation.IP = ipReservation.IP
	err = inv.Get(existingReservation)
	if err != nil && err == inventory.ErrObjectNotFound {
		lambdautils.ErrNotFound()
	} else if err != nil {
		log.Printf("unexpected error getting reservation for '%s': %v", ipReservation.IP, err)
		lambdautils.ErrInternalServerError()
	}

	err = inv.UpdateIPReservation(ipReservation)
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
		return lambdautils.ErrStringResponse(http.StatusBadRequest, "unable to update reservation, the mac may not match the existing reservation or the reservation may no longer exist")
	} else if err != nil {
		log.Printf("error updating reservation: %v", err)
		return lambdautils.ErrInternalServerError()
	}

	subnet, err := lookupSubnetForIP(inv, ipReservation.IP.IP)
	if err != nil {
		log.Printf("Unable to lookup subnet for IP.  This shouldn't happen unless a subnet has been deleted.  %v", err)
		return lambdautils.ErrInternalServerError("Unable to lookup subnet for IP.  This shouldn't happen unless a subnet has been deleted.")
	}

	ipReservation.SetSubnetInformation(subnet)
	return lambdautils.SimpleOKResponse(ipReservation)
}

// DeleteHandler handles POST method requests from the API gateway
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	ipReservation := &types.IPReservation{}
	err := json.Unmarshal([]byte(request.Body), ipReservation)
	if err != nil {
		log.Printf("Unable to parse request: %v", err)
		return lambdautils.ErrBadRequest("Unable to parse request")
	}

	ipAddress := request.PathParameters["ipAddress"]
	ip := net.ParseIP(ipAddress)
	if ip == nil && ipAddress != "" {
		return lambdautils.ErrBadRequest("invalid IP address")
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	err = inv.Delete(ipReservation)
	if err != nil {
		log.Printf("error updating reservation: %v", err)
		return lambdautils.ErrInternalServerError()
	}
	return lambdautils.SimpleOKResponse(nil)
}

// PostHandler handles POST method requests from the API gateway
func PostHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	ipamRequest := &types.IpamIpRequest{}
	err := json.Unmarshal([]byte(request.Body), ipamRequest)
	if err != nil {
		log.Printf("Unable to parse request: %v", err)
		return lambdautils.ErrBadRequest("Unable to parse request")
	}

	ipAddress := request.PathParameters["ipAddress"]
	ip := net.ParseIP(ipAddress)
	if ip == nil && ipAddress != "" {
		return lambdautils.ErrBadRequest("invalid IP address")
	}

	r, err := ipamRequest.Reservation(ip)
	if err != nil {
		log.Printf("got bad request: %v", err)
		return lambdautils.ErrBadRequest(err.Error())
	}

	var subnetLookupIP net.IP
	if ip != nil {
		subnetLookupIP = ip
	} else {
		subnetLookupIP = parseIPOrCidr(ipamRequest.Subnet)
	}

	if subnetLookupIP == nil {
		return lambdautils.ErrBadRequest("provided subnet address is invalid")
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	// Lookup subnet for this request
	subnet, err := lookupSubnetForIP(inv, subnetLookupIP)
	if err != nil {
		log.Printf("unable to lookup subnet for IP %s: %v", r.IP.String(), err)
		return lambdautils.ErrInternalServerError("consult logs for details")
	}

	r.IP = subnet.Cidr

	if ip != nil {
		r.IP.IP = ip

		err = inv.CreateIPReservation(r)
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return lambdautils.ErrStringResponse(http.StatusConflict, "a reservation for this ip address already exists")
		} else if err != nil {
			log.Printf("error creating reservation: %v", err)
			return lambdautils.ErrInternalServerError()
		}

	} else {
		existingReservation, err := getExistingReservationInSubnet(inv, subnet.Cidr, r.MAC)
		if err != nil {
			log.Printf("unexpected error getting existing reservation for %s: %v", r.MAC, err)
			return lambdautils.ErrInternalServerError()
		}

		if existingReservation != nil {
			return lambdautils.ErrStringResponse(http.StatusConflict, "a reservation for this mac address already exists in this subnet")
		}

		for {
			err = r.SetRandomIP()
			if err != nil {
				return lambdautils.ErrInternalServerError()
			}

			err = inv.CreateIPReservation(r)
			if err == nil {
				break
			} else if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != dynamodb.ErrCodeConditionalCheckFailedException {
				log.Printf("error creating reservation: %v", err)
				return lambdautils.ErrInternalServerError()
			}

		}
	}

	r.SetSubnetInformation(subnet)
	return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusCreated, map[string]string{}, r)
}

func parseIPOrCidr(ipString string) net.IP {
	ip := net.ParseIP(ipString)
	if ip != nil {
		return ip
	}

	ip, _, err := net.ParseCIDR(ipString)
	if err == nil {
		return ip
	}
	return nil
}

func getExistingReservationInSubnet(inv *inventory.DynamoDBStore, subnetCidr *net.IPNet, mac net.HardwareAddr) (*types.IPReservation, error) {
	reservations, err := inv.GetIPReservations(subnetCidr)
	if err != nil {
		return nil, err
	}

	for _, r := range reservations {
		if r.MAC.String() == mac.String() {
			return r, nil
		}
	}
	return nil, nil
}

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return GetHandler(ctx, request)
	case http.MethodPost:
		return PostHandler(ctx, request)
	case http.MethodPut:
		return PutHandler(ctx, request)
	case http.MethodDelete:
		return DeleteHandler(ctx, request)
	default:
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotImplemented, map[string]string{}, fmt.Errorf("not implemented"))
	}
}

func main() {
	lambda.Start(Handler)
}
