package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/api/server"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/dynamodbclient"
	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/ipam"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	inv := server.ConnectToInventoryFromContext(ctx)

	if nodeId, ok := request.PathParameters["nodeId"]; ok {
		return server.GetObjectResponse(inv.GetNodeByID(nodeId))
	}

	if len(request.QueryStringParameters) == 0 {
		nodeMap, err := inv.GetNodes()
		nodes := make([]*inventorytypes.Node, 0, len(nodeMap))
		if err == nil {
			for _, n := range nodeMap {
				nodes = append(nodes, n)
			}
		}
		return server.GetObjectResponse(nodes, err)
	}

	if macString, ok := request.QueryStringParameters["mac"]; ok {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return lambdautils.ErrBadRequest(err.Error())
		}

		node, err := inv.GetNodeByMAC(mac)
		return server.GetObjectResponse([]*inventorytypes.Node{node}, err)
	} else if nodeID, ok := request.QueryStringParameters["id"]; ok {
		node, err := inv.GetNodeByID(nodeID)
		return server.GetObjectResponse([]*inventorytypes.Node{node}, err)
	}

	return lambdautils.ErrBadRequest()
}

// PutHandler updates the specified node record
func PutHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	nodeId, ok := request.PathParameters["nodeId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Updating all nodes not allowed.")
	}

	// parse request body.  Should be a node
	updatedNode := &inventorytypes.Node{}
	err := json.Unmarshal([]byte(request.Body), updatedNode)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid node.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	existingNode, err := inv.GetNodeByID(updatedNode.ID())
	if err != nil && err == dynamodbclient.ErrObjectNotFound {
		return lambdautils.ErrNotFound("unable to find existing record for this node")
	} else if err != nil {
		log.Printf("error retrieving existing node %s: %v", updatedNode.ID(), err)
		return lambdautils.ErrInternalServerError()
	}

	for netname, nic := range existingNode.Networks {
		if updatedNic, ok := updatedNode.Networks[netname]; ok &&
			updatedNic.IP.String() == nic.IP.String() &&
			updatedNic.MAC.String() == nic.MAC.String() {
			continue
		}

		network, err := inv.GetNetworkByID(netname)
		if err != nil {
			log.Printf("error getting networks: %v", err)
			return lambdautils.ErrInternalServerError()
		}

		deleteSubnet := network.GetSubnetContainingIP(nic.IP)
		if deleteSubnet == nil {
			continue
		}
		err = inv.Delete(&types.IPReservation{IP: &net.IPNet{IP: nic.IP, Mask: deleteSubnet.Cidr.Mask}})
		if err != nil {
			log.Printf("unable to delete IP for NIC: %v", err)
			return lambdautils.ErrInternalServerError(fmt.Sprintf("unable to delete reservation for ip '%s' for this node, consult logs for more information", nic.IP))
		}
	}

	for netname, nic := range updatedNode.Networks {
		if existingNic, ok := existingNode.Networks[netname]; ok &&
			existingNic.IP.String() == nic.IP.String() &&
			existingNic.MAC.String() == nic.MAC.String() {
			continue
		}

		network, err := inv.GetNetworkByID(netname)
		if err != nil {
			log.Printf("error getting networks: %v", err)
			return lambdautils.ErrInternalServerError()
		}

		reservation, err := generateIPReservation(updatedNode, network)
		if err != nil {
			log.Printf("unexpected error while creating reservation for node '%s' on network '%s': %v", updatedNode.InventoryID, network.ID(), err)
			lambdautils.ErrInternalServerError("unexpected error while creating ip reservation")
		}

		if reservation == nil {
			continue
		}

		err = inv.CreateOrUpdateIPReservation(reservation)
		if err != nil {
			log.Printf("unable to reserve IP for NIC: %v", err)
			return lambdautils.ErrInternalServerError(fmt.Sprintf("unable to reserve ip '%s' for this node, consult logs for more information", reservation.IP))
		}
		nic.IP = reservation.IP.IP
	}

	return server.UpdateObject(inv, updatedNode, nodeId)
}

// PostHandler updates the specified node record
func PostHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	if len(request.PathParameters) != 0 {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Posting not allowed here.")
	}

	// parse request body.  Should be a node
	newNode := &inventorytypes.Node{}
	err := json.Unmarshal([]byte(request.Body), newNode)
	if err != nil {
		return lambdautils.ErrBadRequest("Body should contain a valid node.")
	}

	inv := server.ConnectToInventoryFromContext(ctx)

	for netname, nic := range newNode.Networks {
		if nic.IP == nil && nic.MAC == nil {
			continue
		}

		network, err := inv.GetNetworkByID(netname)
		if err != nil && err == dynamodbclient.ErrObjectNotFound {
			return lambdautils.ErrBadRequest(fmt.Sprintf("%s is not a valid network name", netname))
		} else if err != nil {
			log.Printf("unable to lookup network for nic '%v' on network '%s': %v", nic, netname, err)
			return lambdautils.ErrInternalServerError()
		}

		reservation, err := generateIPReservation(newNode, network)
		if err != nil {
			log.Printf("unexpected error while creating reservation for node '%s' on network '%s': %v", newNode.InventoryID, network.ID(), err)
			lambdautils.ErrInternalServerError("unexpected error while creating ip reservation")
		}

		if reservation == nil {
			continue
		}

		reservation.HostInformation = newNode.InventoryID
		log.Print(reservation)
		err = inv.CreateOrUpdateIPReservation(reservation)
		if err != nil {
			log.Printf("unable to reserve IP for NIC: %v", err)
			return lambdautils.ErrInternalServerError(fmt.Sprintf("unable to reserve ip '%s' for this node, consult logs for more information", reservation.IP))
		}
		nic.IP = reservation.IP.IP

	}

	return server.CreateObject(inv, newNode)
}

func generateIPReservation(node *types.Node, network *types.Network) (*types.IPReservation, error) {
	var ip *net.IPNet
	nic := node.Networks[network.ID()]

	for _, subnet := range network.Subnets {
		if nic.IP == nil {
			allocatedIP, _, _, err := subnet.GetNicConfig(node)
			if err == nil {
				ip = &allocatedIP
				break
			} else if err != ipam.ErrAllocationNotImplemented {
				return nil, fmt.Errorf("unexpected error allocating IP for nic: %v", err)
			}
		} else if subnet.Cidr.Contains(nic.IP) {
			ip = &net.IPNet{
				IP:   nic.IP,
				Mask: subnet.Cidr.Mask,
			}
		}
	}

	if ip == nil {
		return nil, nil
	}

	sTime := time.Now()
	reservation := &types.IPReservation{IP: ip, MAC: nic.MAC, Start: &sTime}
	return reservation, nil
}

// DeleteHandler updates the specified node record
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	nodeId, ok := request.PathParameters["nodeId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Deleting all nodes not allowed.")
	}
	node := &inventorytypes.Node{InventoryID: nodeId}

	inv := server.ConnectToInventoryFromContext(ctx)

	return server.DeleteObject(inv, node)
}

// Handler handles requests for nodes
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return GetHandler(ctx, request)
	case http.MethodPut:
		return PutHandler(ctx, request)
	case http.MethodPost:
		return PostHandler(ctx, request)
	case http.MethodDelete:
		return DeleteHandler(ctx, request)
	default:
		return lambdautils.ErrNotImplemented()
	}
}

func main() {
	lambda.Start(Handler)
}
