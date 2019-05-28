package dynamodbclient

import (
	"errors"

	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// DynamoDBStoreTable defines an interface for describing a dynamodb table
type DynamoDBStoreTable interface {
	GetName() string
	GetKeySchema() []*dynamodb.KeySchemaElement
	GetPartitionKeyName() string
	GetKeyAttributeDefinitions() []*dynamodb.AttributeDefinition
	GetKeyFrom(interface{}) (map[string]*dynamodb.AttributeValue, error)
	GetItemQueryInputFrom(interface{}) (*dynamodb.QueryInput, error)
}

var (
	ErrObjectNotFound = errors.New("Object not found")
)
