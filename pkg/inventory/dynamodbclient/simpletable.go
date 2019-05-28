package dynamodbclient

import (
	"fmt"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type SimpleTableItem interface {
	ID() string
}

type SimpleDynamoDBInventoryTable struct {
	Name string
}

func (t *SimpleDynamoDBInventoryTable) GetName() string {
	return t.Name
}

func (t *SimpleDynamoDBInventoryTable) GetPartitionKeyName() string {
	return "id"
}

func (t *SimpleDynamoDBInventoryTable) GetKeySchema() []*dynamodb.KeySchemaElement {
	return []*dynamodb.KeySchemaElement{
		{
			AttributeName: aws.String("id"),
			KeyType:       aws.String("HASH"),
		},
	}
}

func (t *SimpleDynamoDBInventoryTable) GetKeyAttributeDefinitions() []*dynamodb.AttributeDefinition {
	return []*dynamodb.AttributeDefinition{
		{
			AttributeName: aws.String("id"),
			AttributeType: aws.String("S"),
		},
	}
}

func (t *SimpleDynamoDBInventoryTable) GetKeyFrom(o interface{}) (map[string]*dynamodb.AttributeValue, error) {
	obj, valid := o.(SimpleTableItem)
	if !valid {
		return nil, fmt.Errorf("unsupported object type: %T", o)
	}
	if obj.ID() == "" {
		return nil, types.ErrKeyNotSet
	}
	objID, err := dynamodbattribute.Marshal(obj.ID())
	if err != nil {
		return nil, fmt.Errorf("unable to marshal object id for deletion: %v", err)
	}
	return map[string]*dynamodb.AttributeValue{"id": objID}, nil
}

func (t *SimpleDynamoDBInventoryTable) GetItemQueryInputFrom(o interface{}) (*dynamodb.QueryInput, error) {
	obj, valid := o.(SimpleTableItem)
	if !valid {
		return nil, fmt.Errorf("unsupported object type: %T", o)
	}

	if obj.ID() == "" {
		return nil, types.ErrKeyNotSet
	}

	queryValues, err := dynamodbattribute.MarshalMap(map[string]string{":partitionkeyval": obj.ID()})
	if err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf("%s=:partitionkeyval", "id")
	q := &dynamodb.QueryInput{
		TableName:                 aws.String(t.GetName()),
		KeyConditionExpression:    aws.String(queryString),
		ExpressionAttributeValues: queryValues,
	}
	return q, nil
}
