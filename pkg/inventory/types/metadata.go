package types

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type Metadata map[string]interface{}

func (m *Metadata) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	if av.M != nil {
		md := map[string]interface{}{}
		err := dynamodbattribute.UnmarshalMap(av.M, &md)
		*m = md
		return err
	} else if av.NULL != nil && *av.NULL {
		*m = map[string]interface{}{}
	}
	return nil
}
