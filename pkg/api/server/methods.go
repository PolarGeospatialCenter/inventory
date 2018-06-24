package server

import (
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/PolarGeospatialCenter/inventory/pkg/lambdautils"
	"github.com/aws/aws-lambda-go/events"
)

// InventoryDatabase defines the interface we're expecting for the inventory
type InventoryDatabase interface {
	Exists(inventory.InventoryObject) (bool, error)
	Update(inventory.InventoryObject) error
	Delete(inventory.InventoryObject) error
}

// UpdateObject updates an object
func UpdateObject(inv InventoryDatabase, obj inventory.InventoryObject, id string) (*events.APIGatewayProxyResponse, error) {
	if obj.ID() != id {
		return lambdautils.ErrBadRequest("ID of updated object must match the id specified in the request.")
	}

	exists, err := inv.Exists(obj)
	switch {
	case exists:
		break
	case !exists && err == nil:
		return lambdautils.ErrNotFound(err.Error())
	default:
		return lambdautils.ErrResponse(http.StatusInternalServerError, nil)
	}

	err = inv.Update(obj)
	if err == nil {
		return lambdautils.SimpleOKResponse(obj)
	}

	return lambdautils.ErrInternalServerError()
}

// CreateObject creates an object
func CreateObject(inv InventoryDatabase, obj inventory.InventoryObject) (*events.APIGatewayProxyResponse, error) {
	exists, err := inv.Exists(obj)
	switch {
	case exists:
		return lambdautils.ErrStringResponse(http.StatusConflict, "An object with that id already exists.")
	case !exists && err == nil:
		break
	default:
		return lambdautils.ErrInternalServerError()
	}

	err = inv.Update(obj)
	if err == nil {
		return lambdautils.NewJSONAPIGatewayProxyResponse(http.StatusCreated, map[string]string{}, obj)
	}

	return lambdautils.ErrInternalServerError()
}

// DeleteObject deletes an object
func DeleteObject(inv InventoryDatabase, obj inventory.InventoryObject) (*events.APIGatewayProxyResponse, error) {
	exists, err := inv.Exists(obj)
	switch {
	case exists:
		break
	case !exists && err == nil:
		return lambdautils.ErrNotFound("Objects must exist before you can delete them.")
	default:
		return lambdautils.ErrInternalServerError()
	}

	err = inv.Delete(obj)
	if err == nil {
		return lambdautils.SimpleOKResponse("")
	}

	return lambdautils.ErrInternalServerError()
}

// GetObjectResponse looks up the appropriate response for object
func GetObjectResponse(obj interface{}, err error) (*events.APIGatewayProxyResponse, error) {
	switch err {
	case inventory.ErrObjectNotFound:
		return lambdautils.ErrNotFound(err.Error())
	case nil:
		return lambdautils.SimpleOKResponse(obj)
	default:
		return lambdautils.ErrInternalServerError()
	}
}
