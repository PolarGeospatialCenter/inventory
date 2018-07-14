package types

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/go-test/deep"
)

func TestTagsDynamoDBUnmarshalNil(t *testing.T) {
	av := &dynamodb.AttributeValue{}
	var tags Tags
	err := dynamodbattribute.Unmarshal(av, &tags)
	if err != nil {
		t.Errorf("Unable to unmarshal: %v", err)
	}
	if tags == nil {
		t.Errorf("Got nil, expecting empty slice")
	}
}

func TestTagsDynamoDBUnmarshal(t *testing.T) {
	expectedTags := Tags{"foo", "bar"}
	av, err := dynamodbattribute.Marshal(&expectedTags)
	if err != nil {
		t.Errorf("Unable to marshal tags: %v", err)
	}
	var tags Tags
	err = dynamodbattribute.Unmarshal(av, &tags)
	if err != nil {
		t.Errorf("Unable to unmarshal: %v", err)
	}
	if diff := deep.Equal(tags, expectedTags); len(diff) > 0 {
		t.Errorf("Unmarshaled tags don't match expected:")
		for _, l := range diff {
			t.Error(l)
		}
	}
}
