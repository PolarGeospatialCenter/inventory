package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/azenk/iputils"
)

type IpamIpRequest struct {
	Name      string        `json:"name"`
	Subnet    string        `json:"subnet"`
	HwAddress string        `json:"mac"`
	TTL       time.Duration `json:"ttl"`
}

type IpamIpResponse struct {
	IP      string
	Gateway net.IP
	DNS     []net.IP
	Start   time.Time
	End     time.Time
}

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

	// build IP response
	response := &IpamIpResponse{}
	response.IP = reservation.IP.String()
	if reservation.Start != nil {
		response.Start = *reservation.Start
	}
	if reservation.End != nil {
		response.End = *reservation.End
	}

	response.DNS = subnet.DNS
	response.Gateway = subnet.Gateway

	return lambdautils.SimpleOKResponse(response)
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

// PostHandler handles POST method requests from the API gateway
func PostHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	ipamRequest := &IpamIpRequest{}
	err := json.Unmarshal([]byte(request.Body), ipamRequest)
	if err != nil {
		log.Printf("Unable to parse request: %v", err)
		return lambdautils.ErrBadRequest("Unable to parse request")
	}

	ipAddress := request.PathParameters["ipAddress"]

	if ipamRequest.HwAddress == "" && ipAddress == "" {
		return lambdautils.ErrBadRequest("must specify a mac or IP address to create a reservation")
	}

	var mac net.HardwareAddr
	if ipamRequest.HwAddress != "" {
		mac, err = net.ParseMAC(ipamRequest.HwAddress)
		if err != nil {
			return lambdautils.ErrBadRequest("invalid MAC address")
		}
	}

	ip := net.ParseIP(ipAddress)
	if ip == nil && ipAddress != "" {
		return lambdautils.ErrBadRequest("invalid IP address")
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	r := &types.IPReservation{}

	// if an IP is specified, check for active reservation - if already exists and mac doesn't match, return conflict
	// if no IP is specified, create a new reservation using the mac address
	start := time.Now()
	end := start.Add(ipamRequest.TTL)

	r.Start = &start
	r.End = &end

	if mac != nil {
		r.MAC = mac
	}

	if ip != nil {
		subnet, err := lookupSubnetForIP(inv, ip)
		if err != nil {
			log.Printf("unable to lookup subnet for IP %s: %v", r.IP.String(), err)
			return lambdautils.ErrInternalServerError("consult logs for details")
		}
		log.Printf("Subnet: %v", subnet)
		r.IP = &net.IPNet{IP: ip, Mask: subnet.Cidr.Mask}

		// Try to get it
		res, err := inv.GetIPReservation(r.IP)
		log.Print(res, err)
		if err != nil && err != inventory.ErrObjectNotFound {
			log.Printf("unexpected error while looking up existing reservation: %v", err)
			return lambdautils.ErrInternalServerError()
		} else if err == nil {
			return lambdautils.ErrStringResponse(http.StatusConflict, "a reservation for this ip already exists")
		}
	}

	if r.IP == nil && mac != nil {
		// don't have an IP yet, but we have a mac, let's try to look up a matching node
		node, err := inv.GetNodeByMAC(mac)
		if err == nil {
			// Found a node
			for _, network := range node.Networks {
				if network.MAC != nil && mac.String() == network.MAC.String() && network.IP != nil {
					subnet, err := lookupSubnetForIP(inv, network.IP)
					if err != nil {
						log.Printf("unable to lookup subnet for IP %s: %v", r.IP.String(), err)
						return lambdautils.ErrInternalServerError("consult logs for details")
					}
					log.Printf("Subnet: %v", subnet)
					r.IP = &net.IPNet{IP: network.IP, Mask: subnet.Cidr.Mask}
					break
				}
			}
		}
	}

	if ipamRequest.Subnet != "" {
		subnetIP := net.ParseIP(ipamRequest.Subnet)
		if subnetIP == nil {
			subnetIP, _, err = net.ParseCIDR(ipamRequest.Subnet)
			if err != nil {
				return lambdautils.ErrBadRequest("provided subnet address is invalid")
			}
		}

		subnet, err := lookupSubnetForIP(inv, subnetIP)
		if err != nil {
			log.Printf("unable to lookup subnet for IP %s: %v", r.IP.String(), err)
			return lambdautils.ErrNotFound("unable to find matching subnet, please update your request and try again")
		}

		startOffset, ipLength := subnet.Cidr.Mask.Size()
		networkIP, err := iputils.SetBits(subnet.Cidr.IP, uint64(0), uint(startOffset), uint(ipLength-startOffset))
		broadcastIP, err := iputils.SetBits(subnet.Cidr.IP, uint64(0xffffffffffffffff), uint(startOffset), uint(ipLength-startOffset))

		for r.IP == nil {
			// choose IP at random until we find a free one
			randomHostPart := rand.Uint64()
			candidateIP, err := iputils.SetBits(subnet.Cidr.IP, randomHostPart, uint(startOffset), uint(ipLength-startOffset))
			if err != nil {
				log.Printf("unexpected error building ip: %v", err)
				return lambdautils.ErrInternalServerError()
			}
			if candidateIP.To4() != nil && (candidateIP.Equal(networkIP) || candidateIP.Equal(broadcastIP)) {
				continue
			}
			r.IP = &net.IPNet{IP: candidateIP, Mask: subnet.Cidr.Mask}
			_, err = inv.GetIPReservation(r.IP)
			if err != nil && err != inventory.ErrObjectNotFound {
				log.Printf("unexpected error while looking up existing reservation: %v", err)
				return lambdautils.ErrInternalServerError()
			} else if err == nil {
				r.IP = nil
			}
		}

	}
	err = inv.CreateIPReservation(r)
	if err != nil {
		log.Printf("error creating reservation: %v", err)
		return lambdautils.ErrInternalServerError()
	}

	return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusCreated, map[string]string{}, r)
}

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return GetHandler(ctx, request)
	case http.MethodPost:
		return PostHandler(ctx, request)
	default:
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotImplemented, map[string]string{}, fmt.Errorf("not implemented"))
	}
}

func main() {
	lambda.Start(Handler)
}
