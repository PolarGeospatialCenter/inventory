package dynamodbclient

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/azenk/iputils"
)

type IPReservationStore struct {
	*DynamoDBStore
}

func (db *IPReservationStore) GetIPReservation(ipNet *net.IPNet) (*types.IPReservation, error) {
	r := &types.IPReservation{
		IP: ipNet,
	}
	err := db.get(r)
	return r, err
}

func (db *IPReservationStore) GetIPReservationsByMac(mac net.HardwareAddr) (types.IPReservationList, error) {
	allReservations := make(types.IPReservationList, 0)
	err := db.getAll(&allReservations)
	if err != nil {
		return nil, err
	}
	result := make(types.IPReservationList, 0)
	for _, reservation := range allReservations {
		if reservation.MAC.String() == mac.String() {
			result = append(result, reservation)
		}
	}
	return result, nil
}

// GetIPReservations returns all current reservations in the specified subnet
func (db *IPReservationStore) GetIPReservations(ipNet *net.IPNet) (types.IPReservationList, error) {
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

	out := make(types.IPReservationList, len(results.Items))

	err = dynamodbattribute.UnmarshalListOfMaps(results.Items, &out)

	return out, err
}

func (db *IPReservationStore) GetExistingIPReservationInSubnet(subnetCidr *net.IPNet, mac net.HardwareAddr) (*types.IPReservation, error) {
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

func (db *IPReservationStore) CreateRandomIPReservation(r *types.IPReservation, subnet *types.Subnet) (*types.IPReservation, error) {
	maxCount := 10
	reservation := *r
	for count := 0; count < maxCount; count++ {
		existingReservations, err := db.GetIPReservations(subnet.Cidr)
		if err != nil {
			return nil, err
		}

		startOffset, ipLength := subnet.Cidr.Mask.Size()
		if len(existingReservations) >= (1<<uint(ipLength-startOffset) - 2) {
			return nil, fmt.Errorf("this subnet is full, cannot allocate an address")
		}

		rand.Seed(time.Now().UnixNano())

		reservation.IP = &net.IPNet{Mask: subnet.Cidr.Mask}
		if reservation.Start == nil {
			start := time.Now()
			reservation.Start = &start
		}

		// generate random IP in the subnet
		// Check to see if reservation list contains a reservation for it
		// if it's in the list of reserved addresses, try again
		// if it's not in the list of addresses, try to reserve it
		// if reservation fails, retry up to N times?  or just error?
		for {
			// choose IP at random until we find a free one
			randomHostPart := rand.Uint64()
			candidateIP, err := iputils.SetBits(subnet.Cidr.IP, randomHostPart, uint(startOffset), uint(ipLength-startOffset))
			if err != nil {
				return nil, fmt.Errorf("unexpected error building ip: %v", err)
			}
			reservation.IP.IP = candidateIP

			if !reservation.Validate() || existingReservations.Contains(candidateIP) {
				continue
			}

			err = db.CreateIPReservation(&reservation)
			if err != nil && err == ErrAlreadyExists {
				break
			}
			return &reservation, err
		}
	}
	return nil, fmt.Errorf("retry limit exceeded: giving up on reserving an ip for %v", r)
}

func (db *IPReservationStore) CreateIPReservation(r *types.IPReservation) error {
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

func (db *IPReservationStore) UpdateIPReservation(r *types.IPReservation) error {
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

func (db *IPReservationStore) CreateOrUpdateIPReservation(r *types.IPReservation) error {
	err := db.UpdateIPReservation(r)
	if err == nil {
		return nil
	}

	if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != dynamodb.ErrCodeConditionalCheckFailedException {
		return err
	}

	return db.CreateIPReservation(r)
}

func (db *IPReservationStore) Exists(r *types.IPReservation) (bool, error) {
	return db.DynamoDBStore.exists(r)
}

func (db *IPReservationStore) Delete(r *types.IPReservation) error {
	return db.DynamoDBStore.delete(r)
}

func (db *IPReservationStore) ObjExists(obj interface{}) (bool, error) {
	r, ok := obj.(*types.IPReservation)
	if !ok {
		return false, ErrInvalidObjectType
	}
	return db.Exists(r)
}

func (db *IPReservationStore) ObjCreate(obj interface{}) error {
	r, ok := obj.(*types.IPReservation)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.CreateIPReservation(r)
}

func (db *IPReservationStore) ObjUpdate(obj interface{}) error {
	r, ok := obj.(*types.IPReservation)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.UpdateIPReservation(r)
}

func (db *IPReservationStore) ObjDelete(obj interface{}) error {
	r, ok := obj.(*types.IPReservation)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Delete(r)
}
