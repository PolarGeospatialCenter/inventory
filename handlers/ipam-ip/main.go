package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/ipam"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type IpamIpRequest struct {
	Requestor string        `json:"requestor"`
	TTL       time.Duration `json:"ttl"`
}

type IpamIpResponse struct {
	IP      net.IP
	Mask    int
	Gateway net.IP
}

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	networkId, ok := request.PathParameters["networkId"]
	if !ok {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, fmt.Errorf("You must specify a network ID"))
	}

	subnetName, ok := request.PathParameters["subnetName"]
	if !ok {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, fmt.Errorf("You must specify a network ID"))
	}

	requestData := &IpamIpRequest{}
	err := json.Unmarshal([]byte(request.Body), requestData)
	if err != nil {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, fmt.Errorf("Unable to parse request body, please fix and resubmit"))
	}

	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	inv := inventory.NewDynamoDBStore(db, nil)

	// lookup network and subnet
	network, err := inv.GetNetworkByID(networkId)
	if err != nil {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotFound, map[string]string{}, fmt.Errorf("You must specify a valid network ID"))
	}

	var subnet *inventorytypes.Subnet
	for _, s := range network.Subnets {
		if s.Name == subnetName {
			subnet = s
			break
		}
	}
	if subnet == nil {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotFound, map[string]string{}, fmt.Errorf("You must specify a valid subnet name"))
	}

	node, err := inv.GetNodeByID(requestData.Requestor)
	if err != nil {
		log.Printf("Error looking up node: %v", err)
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusBadRequest, map[string]string{}, fmt.Errorf("The node you specified could not be found"))
	}

	ip, _, gateway, err := subnet.GetNicConfig(node)
	if err == ipam.ErrAllocationNotImplemented {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotImplemented, map[string]string{}, err)
	} else if err != nil {
		log.Printf("Unable to allocate ip: %v", err)
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusInternalServerError, map[string]string{}, fmt.Errorf("unable to allocate ip"))
	}

	maskSize, _ := ip.Mask.Size()
	response := &IpamIpResponse{IP: ip.IP, Mask: maskSize, Gateway: gateway}
	return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusCreated, map[string]string{}, response)
}

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodPost:
		return GetHandler(ctx, request)
	default:
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusNotImplemented, map[string]string{}, fmt.Errorf("not implemented"))
	}
}

func main() {
	lambda.Start(Handler)
}
