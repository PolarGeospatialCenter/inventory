package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/dynamodbclient"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// InventoryDatabase defines the interface we're expecting for the inventory
type InventoryDatabase interface {
	ObjExists(interface{}) (bool, error)
	ObjUpdate(interface{}) error
	ObjDelete(interface{}) error
	ObjCreate(interface{}) error
}

type InventoryObject interface {
	ID() string
	Timestamp() int64
	SetTimestamp(time.Time)
}

// ConnectToInventoryFromContext creates a dynamodb inventory client from credentials attached to the context
func ConnectToInventoryFromContext(ctx context.Context) *dynamodbclient.DynamoDBStore {
	db := dynamodb.New(lambdautils.AwsContextConfigProvider(ctx))
	return dynamodbclient.NewDynamoDBStore(db, nil)
}

// UpdateObject updates an object
func UpdateObject(inv InventoryDatabase, obj InventoryObject, id string) (*events.APIGatewayProxyResponse, error) {
	if obj.ID() != id {
		return lambdautils.ErrBadRequest("ID of updated object must match the id specified in the request.")
	}

	exists, err := inv.ObjExists(obj)
	switch {
	case exists:
		break
	case !exists && err == nil:
		return lambdautils.ErrNotFound()
	default:
		return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
	}

	if obj.Timestamp() == (&time.Time{}).Unix() {
		obj.SetTimestamp(time.Now())
	}

	err = inv.ObjUpdate(obj)
	if err == nil {
		return lambdautils.SimpleOKResponse(obj)
	}

	return lambdautils.ErrInternalServerError()
}

// CreateObject creates an object
func CreateObject(inv InventoryDatabase, obj InventoryObject) (*events.APIGatewayProxyResponse, error) {
	exists, err := inv.ObjExists(obj)
	switch {
	case exists:
		return lambdautils.ErrStringResponse(http.StatusConflict, "An object with that id already exists.")
	case !exists && err == nil:
		break
	default:
		return lambdautils.ErrInternalServerError()
	}

	if obj.Timestamp() == (&time.Time{}).Unix() {
		obj.SetTimestamp(time.Now())
	}

	err = inv.ObjCreate(obj)
	if err == nil {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusCreated, map[string]string{}, obj)
	}

	return lambdautils.ErrInternalServerError()
}

// DeleteObject deletes an object
func DeleteObject(inv InventoryDatabase, obj InventoryObject) (*events.APIGatewayProxyResponse, error) {
	exists, err := inv.ObjExists(obj)
	switch {
	case exists:
		break
	case !exists && err == nil:
		return lambdautils.ErrNotFound("Objects must exist before you can delete them.")
	default:
		return lambdautils.ErrInternalServerError()
	}

	err = inv.ObjDelete(obj)
	if err == nil {
		return lambdautils.SimpleOKResponse("")
	}

	return lambdautils.ErrInternalServerError()
}

// GetObjectResponse looks up the appropriate response for object
func GetObjectResponse(obj interface{}, err error) (*events.APIGatewayProxyResponse, error) {
	switch err {
	case dynamodbclient.ErrObjectNotFound:
		return lambdautils.ErrNotFound(err.Error())
	case nil:
		return lambdautils.SimpleOKResponse(obj)
	default:
		log.Printf("Returning internal server error.  Actual error was: %v", err)
		return lambdautils.ErrInternalServerError()
	}
}
