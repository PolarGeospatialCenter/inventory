package dynamodbclient

import (
	"fmt"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type IPReservationTable struct {
	Name string
}

func (t *IPReservationTable) GetName() string {
	return t.Name
}

func (t *IPReservationTable) GetKeySchema() []*dynamodb.KeySchemaElement {
	return []*dynamodb.KeySchemaElement{
		{
			AttributeName: aws.String("net"),
			KeyType:       aws.String("HASH"),
		},
		{
			AttributeName: aws.String("ip"),
			KeyType:       aws.String("RANGE"),
		},
	}
}

func (t *IPReservationTable) GetKeyAttributeDefinitions() []*dynamodb.AttributeDefinition {
	return []*dynamodb.AttributeDefinition{
		{
			AttributeName: aws.String("net"),
			AttributeType: aws.String("B"),
		},
		{
			AttributeName: aws.String("ip"),
			AttributeType: aws.String("B"),
		},
	}
}

func (t *IPReservationTable) CreateTable() error {
	return fmt.Errorf("not implemented")
}

func (t *IPReservationTable) GetKeyFrom(o interface{}) (map[string]*dynamodb.AttributeValue, error) {
	obj, valid := o.(*types.IPReservation)
	if !valid {
		return nil, fmt.Errorf("unsupported object type: %T", o)
	}

	if obj.IP == nil {
		return nil, types.ErrKeyNotSet
	}

	net := obj.IP.IP.Mask(obj.IP.Mask)
	if v4IP := net.To4(); v4IP != nil {
		net = v4IP
	}
	netValue, err := dynamodbattribute.Marshal(net)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal object id for deletion: %v", err)
	}

	ip := obj.IP.IP
	if v4IP := obj.IP.IP.To4(); v4IP != nil {
		ip = v4IP
	}
	ipValue, err := dynamodbattribute.Marshal(ip)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal object id for deletion: %v", err)
	}

	return map[string]*dynamodb.AttributeValue{"net": netValue, "ip": ipValue}, nil
}

func (t *IPReservationTable) GetItemQueryInputFrom(o interface{}) (*dynamodb.QueryInput, error) {
	obj, valid := o.(*types.IPReservation)
	if !valid {
		return nil, fmt.Errorf("unsupported object type: %T", o)
	}

	if obj.IP == nil {
		return nil, types.ErrKeyNotSet
	}

	net := obj.IP.IP.Mask(obj.IP.Mask)
	if v4IP := net.To4(); v4IP != nil {
		net = v4IP
	}
	netValue, err := dynamodbattribute.Marshal(net)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal object id for deletion: %v", err)
	}

	ip := obj.IP.IP
	if v4IP := obj.IP.IP.To4(); v4IP != nil {
		ip = v4IP
	}
	ipValue, err := dynamodbattribute.Marshal(ip)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal object id for deletion: %v", err)
	}

	queryValues := map[string]*dynamodb.AttributeValue{":partitionkeyval": netValue, ":rangekeyval": ipValue}

	queryString := "net=:partitionkeyval AND ip=:rangekeyval"
	q := &dynamodb.QueryInput{
		TableName:                 aws.String(t.GetName()),
		KeyConditionExpression:    aws.String(queryString),
		ExpressionAttributeValues: queryValues,
	}
	return q, nil
}

func (t *IPReservationTable) GetPartitionKeyName() string {
	return "net"
}
