package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/PolarGeospatialCenter/inventory/pkg/api/server"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
)

// GetHandler handles GET method requests from the API gateway
func GetHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	inv := server.ConnectToInventoryFromContext(ctx)

	if nodeId, ok := request.PathParameters["nodeId"]; ok {
		return server.GetObjectResponse(inv.Node().GetNodeByID(nodeId))
	}

	if len(request.QueryStringParameters) == 0 {
		nodeMap, err := inv.Node().GetNodes()
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

		node, err := inv.Node().GetNodeByMAC(mac)
		return server.GetObjectResponse([]*inventorytypes.Node{node}, err)
	} else if nodeID, ok := request.QueryStringParameters["id"]; ok {
		node, err := inv.Node().GetNodeByID(nodeID)
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

	sendUpdateEvent(ctx)

	return server.UpdateObject(inv.Node(), updatedNode, nodeId)
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

	sendUpdateEvent(ctx)

	return server.CreateObject(inv.Node(), newNode)
}

// DeleteHandler updates the specified node record
func DeleteHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	nodeId, ok := request.PathParameters["nodeId"]
	if !ok {
		return lambdautils.ErrStringResponse(http.StatusMethodNotAllowed, "Deleting all nodes not allowed.")
	}
	node := &inventorytypes.Node{InventoryID: nodeId}

	inv := server.ConnectToInventoryFromContext(ctx)
	sendUpdateEvent(ctx)

	return server.DeleteObject(inv.Node(), node)
}

func sendUpdateEvent(ctx context.Context) {
	snsClient := server.ConnectToSNSFromContext(ctx)
	var topicArn string
	err := snsClient.ListTopicsPages(&sns.ListTopicsInput{}, func(out *sns.ListTopicsOutput, last bool) bool {
		for _, topic := range out.Topics {
			if topic.TopicArn == nil {
				continue
			}
			arnString := *topic.TopicArn
			if strings.HasSuffix(arnString, ":inventory_node_events") {
				topicArn = arnString
				return false
			}
		}
		return !last
	})
	if err != nil {
		log.Printf("unable to list sns topics: %v", err)
	}

	if topicArn == "" {
		log.Printf("no SNS topic found")
		return
	}

	_, err = snsClient.Publish(&sns.PublishInput{
		Message:  aws.String("{}"),
		TopicArn: aws.String(topicArn),
	})
	if err != nil {
		log.Printf("unable to publish update to SNS queue '%s': %v", topicArn, err)
	}
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
