package dynamodbclient

import (
	"fmt"
	"net"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/ipam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func (db *DynamoDBStore) generateIPReservation(node *types.Node, network *types.Network) (*types.IPReservation, error) {
	var ip *net.IPNet
	nic := node.Networks[network.ID()]

	for _, subnet := range network.Subnets {
		if nic.IP == nil {
			allocatedIP, _, _, err := subnet.GetNicConfig(node)
			if err == nil {
				ip = &allocatedIP
				break
			} else if err != ipam.ErrAllocationNotImplemented {
				return nil, fmt.Errorf("unexpected error allocating IP for nic: %v", err)
			}
		} else if subnet.Cidr.Contains(nic.IP) {
			ip = &net.IPNet{
				IP:   nic.IP,
				Mask: subnet.Cidr.Mask,
			}
		}
	}

	if ip == nil {
		return nil, nil
	}

	sTime := time.Now()
	reservation := &types.IPReservation{IP: ip, MAC: nic.MAC, Start: &sTime}
	return reservation, nil
}

func (db *DynamoDBStore) GetIPReservation(ipNet *net.IPNet) (*types.IPReservation, error) {
	r := &types.IPReservation{
		IP: ipNet,
	}
	err := db.Get(r)
	return r, err
}

// GetIPReservations returns all current reservations in the specified subnet
func (db *DynamoDBStore) GetIPReservations(ipNet *net.IPNet) ([]*types.IPReservation, error) {
	table := db.tableMap.LookupTable(&types.IPReservation{})
	if table == nil {
		return nil, fmt.Errorf("No table found for object of type %T", &types.IPReservation{})
	}

	if ipNet == nil {
		return nil, fmt.Errorf("specified network is nil")
	}

	netValue, err := dynamodbattribute.Marshal(ipNet.IP.Mask(ipNet.Mask))
	if err != nil {
		return nil, fmt.Errorf("unable to marshal object id for deletion: %v", err)
	}

	queryValues := map[string]*dynamodb.AttributeValue{":partitionkeyval": netValue}

	queryString := "net=:partitionkeyval"
	q := &dynamodb.QueryInput{
		TableName:                 aws.String(table.GetName()),
		KeyConditionExpression:    aws.String(queryString),
		ExpressionAttributeValues: queryValues,
	}

	results, err := db.db.Query(q)
	if err != nil {
		return nil, err
	}

	out := make([]*types.IPReservation, len(results.Items))

	err = dynamodbattribute.UnmarshalListOfMaps(results.Items, &out)

	return out, err
}

func (db *DynamoDBStore) GetExistingIPReservationInSubnet(subnetCidr *net.IPNet, mac net.HardwareAddr) (*types.IPReservation, error) {
	reservations, err := db.GetIPReservations(subnetCidr)
	if err != nil {
		return nil, err
	}

	for _, r := range reservations {
		if r.MAC.String() == mac.String() {
			return r, nil
		}
	}
	return nil, nil
}

func (db *DynamoDBStore) CreateIPReservation(r *types.IPReservation) error {
	table := db.tableMap.LookupTable(r)
	if table == nil {
		return fmt.Errorf("No table found for object of type %T", r)
	}
	putItem := &dynamodb.PutItemInput{}
	putItem.SetTableName(table.GetName())
	item, err := dynamodbattribute.MarshalMap(r)
	if err != nil {
		return err
	}
	putItem.Item = item

	keyMap, err := table.GetKeyFrom(r)
	if err != nil {
		return err
	}

	for k, v := range keyMap {
		putItem.Item[k] = v
	}

	putItem.SetConditionExpression("attribute_not_exists(net) and attribute_not_exists(ip)")
	_, err = db.db.PutItem(putItem)
	return err
}

func (db *DynamoDBStore) UpdateIPReservation(r *types.IPReservation) error {
	table := db.tableMap.LookupTable(r)
	if table == nil {
		return fmt.Errorf("No table found for object of type %T", r)
	}
	putItem := &dynamodb.PutItemInput{}
	putItem.SetTableName(table.GetName())
	item, err := dynamodbattribute.MarshalMap(r)
	if err != nil {
		return err
	}
	putItem.Item = item

	keyMap, err := table.GetKeyFrom(r)
	if err != nil {
		return err
	}

	for k, v := range keyMap {
		putItem.Item[k] = v
	}

	putItem.SetConditionExpression("net = :net and ip = :ip and MAC = :mac")
	macAddress, err := dynamodbattribute.Marshal(r.MAC.String())
	if err != nil {
		return err
	}
	keyAttributes, err := table.GetKeyFrom(r)

	putItem.SetExpressionAttributeValues(map[string]*dynamodb.AttributeValue{":mac": macAddress, ":net": keyAttributes["net"], ":ip": keyAttributes["ip"]})
	_, err = db.db.PutItem(putItem)
	return err
}

func (db *DynamoDBStore) CreateOrUpdateIPReservation(r *types.IPReservation) error {
	err := db.UpdateIPReservation(r)
	if err == nil {
		return nil
	}

	if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != dynamodb.ErrCodeConditionalCheckFailedException {
		return err
	}

	return db.CreateIPReservation(r)
}
