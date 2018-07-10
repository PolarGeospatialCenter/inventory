package types

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type Tags []string

func (t *Tags) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	if av.L != nil {
		l := []string{}
		err := dynamodbattribute.UnmarshalList(av.L, &l)
		if err != nil {
			return err
		}
		*t = l
		return nil
	}
	*t = Tags{}
	return nil
}
