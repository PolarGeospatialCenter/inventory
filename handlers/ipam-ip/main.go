package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/api/server"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/dynamodbclient"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	ip, gotIP := request.PathParameters["ipAddress"]
	macQuery, gotMAC := request.QueryStringParameters["mac"]
	if !gotIP && !gotMAC {
		return lambdautils.ErrBadRequest("You must specify a mac query or an IP address")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	if gotIP {
		ipAddress := net.ParseIP(ip)
		if ipAddress == nil {
			return lambdautils.ErrBadRequest("Bad IP address")
		}

		// lookup network and subnet
		subnet, err := lookupSubnetForIP(inv, ipAddress)
		if err != nil {
			log.Printf("unable to lookup subnet for IP %s: %v", ipAddress, err)
			return lambdautils.ErrInternalServerError("consult logs for details")
		}

		reservation, err := inv.IPReservation().GetIPReservation(&net.IPNet{IP: ipAddress, Mask: subnet.Cidr.Mask})
		if err != nil {
			return lambdautils.ErrNotFound("No reservation found for that IP")
		}

		reservation.SetSubnetInformation(subnet)
		return lambdautils.SimpleOKResponse(reservation)
	}

	if gotMAC {
		mac, err := net.ParseMAC(macQuery)
		if err != nil {
			return lambdautils.ErrBadRequest("Bad MAC address")
		}

		reservations, err := inv.IPReservation().GetIPReservationsByMac(mac)
		if err != nil {
			log.Printf("Unable to lookup reservations for mac address '%s': %v", mac.String(), err)
			return lambdautils.ErrInternalServerError()
		}

		for _, r := range reservations {
			subnet, err := lookupSubnetForIP(inv, r.IP.IP)
			if err != nil {
				log.Printf("error looking up subnet for ip reservation: %v", err)
				return lambdautils.ErrInternalServerError()
			}
			r.SetSubnetInformation(subnet)
		}
		return lambdautils.SimpleOKResponse(reservations)

	}

	log.Printf("Unknown issue getting ip reservations.  This shouldn't happen.")
	return lambdautils.ErrInternalServerError()

}

func lookupSubnetForIP(inv *dynamodbclient.DynamoDBStore, ip net.IP) (*types.Subnet, error) {
	networks, err := inv.Network().GetNetworks()
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

// PutHandler handles PUT method requests from the API gateway
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

	inv := server.ConnectToInventoryFromContext(ctx)

	subnet, err := lookupSubnetForIP(inv, ip)
	if err != nil {
		log.Printf("unable to lookup subnet for IP %s: %v", ipAddress, err)
		return lambdautils.ErrInternalServerError("consult logs for details")
	}
	ipReservation.IP = &net.IPNet{IP: ip, Mask: subnet.Cidr.Mask}

	_, err = inv.IPReservation().GetIPReservation(ipReservation.IP)
	if err != nil && err == dynamodbclient.ErrObjectNotFound {
		lambdautils.ErrNotFound()
	} else if err != nil {
		log.Printf("unexpected error getting reservation for '%s': %v", ipReservation.IP, err)
		lambdautils.ErrInternalServerError()
	}

	err = inv.IPReservation().UpdateIPReservation(ipReservation)
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
		return lambdautils.ErrStringResponse(http.StatusBadRequest, "unable to update reservation, the mac may not match the existing reservation or the reservation may no longer exist")
	} else if err != nil {
		log.Printf("error updating reservation: %v", err)
		return lambdautils.ErrInternalServerError()
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

	inv := server.ConnectToInventoryFromContext(ctx)

	subnet, err := lookupSubnetForIP(inv, ip)
	if err != nil {
		log.Printf("unable to lookup subnet for IP %s: %v", ipAddress, err)
		return lambdautils.ErrInternalServerError("consult logs for details")
	}
	ipReservation.IP = &net.IPNet{IP: ip, Mask: subnet.Cidr.Mask}

	err = inv.IPReservation().Delete(ipReservation)
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

	inv := server.ConnectToInventoryFromContext(ctx)

	// Lookup subnet for this request
	subnet, err := lookupSubnetForIP(inv, subnetLookupIP)
	if err != nil {
		log.Printf("unable to lookup subnet for IP %s: %v", r.IP.String(), err)
		return lambdautils.ErrInternalServerError("consult logs for details")
	}

	existingReservation, err := inv.IPReservation().GetExistingIPReservationInSubnet(subnet.Cidr, r.MAC)
	if err != nil {
		log.Printf("unexpected error getting existing reservation for %s: %v", r.MAC, err)
		return lambdautils.ErrInternalServerError()
	}

	if r.MAC.String() != "" && existingReservation != nil {
		return lambdautils.ErrStringResponse(http.StatusConflict, "a reservation for this mac address already exists in this subnet")
	}

	r.IP = subnet.Cidr

	if ip != nil {
		r.IP.IP = ip

		err = inv.IPReservation().CreateIPReservation(r)
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return lambdautils.ErrStringResponse(http.StatusConflict, "a reservation for this ip address already exists")
		} else if err != nil {
			log.Printf("error creating reservation: %v", err)
			return lambdautils.ErrInternalServerError()
		}

	} else if subnet.DynamicAllocationEnabled() {
		r, err = inv.IPReservation().CreateRandomIPReservation(r, subnet)
		if err != nil {
			return lambdautils.ErrInternalServerError()
		}
	} else {
		return lambdautils.ErrBadRequest("unable to allocate an IP in the requested subnet")
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
