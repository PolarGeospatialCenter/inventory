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

// GetString attempts to get a string value with the provided key.
func (m Metadata) GetString(key string) (string, bool) {
	if iVal, ok := m[key]; ok {
		val, sok := iVal.(string)
		return val, sok
	}
	return "", false
}
