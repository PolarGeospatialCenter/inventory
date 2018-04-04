package inventory

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type RawInventoryStore interface {
	GetNodes() (map[string]*types.Node, error)
	GetNetworks() (map[string]*types.Network, error)
	GetSystems() (map[string]*types.System, error)
}

// DynamoDBStoreTableMap maps data types to the appropriate table within DynamoDB
type DynamoDBStoreTableMap struct {
	Node         string
	Network      string
	System       string
	NodeMacIndex string
}

func (m *DynamoDBStoreTableMap) LookupTable(t interface{}) string {
	switch t.(type) {
	case *types.Node:
		return m.Node
	case *types.Network:
		return m.Network
	case *types.System:
		return m.System
	case *NodeMacIndexEntry:
		return m.NodeMacIndex
	}
	return ""
}

func (m *DynamoDBStoreTableMap) Tables() []string {
	return []string{m.Node, m.Network, m.System, m.NodeMacIndex}
}

var (
	defatultDynamoDBTables = &DynamoDBStoreTableMap{
		Node:         "inventory_nodes",
		Network:      "inventory_networks",
		System:       "inventory_systems",
		NodeMacIndex: "inventory_node_mac_lookup",
	}
)

type NodeMacIndexEntry struct {
	Mac         net.HardwareAddr
	LastUpdated time.Time
	NodeID      string
}

func (i *NodeMacIndexEntry) ID() string {
	return i.Mac.String()
}

func (i *NodeMacIndexEntry) Timestamp() int64 {
	return i.LastUpdated.Unix()
}

type DynamoDBTableLookup interface {
	LookupTable(interface{}) string
	Tables() []string
}

type ErrDynamoDBRecordNotFound struct {
	ID    string
	Table string
}

func (e ErrDynamoDBRecordNotFound) Error() string {
	return fmt.Sprintf("no items returned for id %s in table %s", e.ID, e.Table)
}

type DynamoDBStore struct {
	tableMap DynamoDBTableLookup
	db       *dynamodb.DynamoDB
}

// NewDynamoDBStore creates a DynamoDBStore
func NewDynamoDBStore(db *dynamodb.DynamoDB, tableMap DynamoDBTableLookup) (*DynamoDBStore, error) {
	if tableMap == nil {
		tableMap = defatultDynamoDBTables
	}
	obj := &DynamoDBStore{tableMap: tableMap, db: db}
	for _, table := range tableMap.Tables() {
		_, err := db.DescribeTable(&dynamodb.DescribeTableInput{TableName: &table})
		if err == nil {
			continue
		}
		err = obj.createTable(table)
		if err != nil {
			return nil, err
		}
	}
	return obj, err
}

func (db *DynamoDBStore) createTable(table string) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("last_update"),
				AttributeType: aws.String("N"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("last_update"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		TableName: aws.String(table),
	}

	_, err := db.db.CreateTable(input)
	return err
}

func (db *DynamoDBStore) Nodes() (map[string]*types.InventoryNode, error) {
	return map[string]*types.InventoryNode{}, nil
}

func (db *DynamoDBStore) Refresh() error {
	return nil
}

func (db *DynamoDBStore) Update(obj InventoryObject) error {
	putItem := &dynamodb.PutItemInput{}
	putItem.SetTableName(db.tableMap.LookupTable(obj))

	switch obj.(type) {
	case *types.Node:
		node := obj.(*types.Node)
		putItem.Item, _ = dynamodbattribute.MarshalMap(node)
		for _, nic := range node.Networks {
			if nic.MAC != nil {
				db.Update(&NodeMacIndexEntry{Mac: nic.MAC, LastUpdated: node.LastUpdated, NodeID: node.ID()})
			}
		}
	case *types.Network:
		network := obj.(*types.Network)
		putItem.Item, _ = dynamodbattribute.MarshalMap(network)
	case *types.System:
		system := obj.(*types.System)
		putItem.Item, _ = dynamodbattribute.MarshalMap(system)
	case *NodeMacIndexEntry:
		e := obj.(*NodeMacIndexEntry)
		putItem.Item, _ = dynamodbattribute.MarshalMap(e)
	default:
		return fmt.Errorf("No matching type for update")
	}

	invObj := obj.(InventoryObject)
	putItem.Item["id"], _ = dynamodbattribute.Marshal(invObj.ID())
	putItem.Item["last_update"], _ = dynamodbattribute.Marshal(invObj.Timestamp())

	log.Print(putItem.TableName)
	_, err := db.db.PutItem(putItem)
	if err != nil {
		return err
	}
	return nil
}

func (db *DynamoDBStore) Delete(interface{}) error {
	return nil
}

func (db *DynamoDBStore) getPartitionKey(table string) (string, error) {
	out, err := db.db.DescribeTable(&dynamodb.DescribeTableInput{TableName: &table})
	if err != nil {
		return "", err
	}
	for _, key := range out.Table.KeySchema {
		if *key.KeyType == "HASH" {
			return *key.AttributeName, nil
		}
	}
	return "", fmt.Errorf("no partition key found for table %s", table)
}

// getNewest returns the entry from the table with a partition id matching id and
// the highest sort key (last_updated timestamp)
func (db *DynamoDBStore) getNewest(id string, out interface{}) error {
	f := false
	table := db.tableMap.LookupTable(out)
	partitionKeyName, err := db.getPartitionKey(table)
	if err != nil {
		return err
	}

	queryValues, err := dynamodbattribute.MarshalMap(map[string]string{":partitionkeyval": id})
	if err != nil {
		return err
	}

	queryString := fmt.Sprintf("%s=:partitionkeyval", partitionKeyName)
	q := &dynamodb.QueryInput{
		ScanIndexForward:          &f,
		TableName:                 &table,
		KeyConditionExpression:    &queryString,
		ExpressionAttributeValues: queryValues,
	}

	results, err := db.db.Query(q)
	if err != nil {
		return err
	}

	if len(results.Items) == 0 {
		return ErrDynamoDBRecordNotFound{ID: id, Table: table}
	}
	err = dynamodbattribute.UnmarshalMap(results.Items[0], out)
	return err
}

func (db *DynamoDBStore) GetInventoryNodeByID(id string) (*types.InventoryNode, error) {
	node, err := db.GetNodeByID(id)
	if err != nil {
		return nil, err
	}

	return types.NewInventoryNode(node, db, db)
}

func (db *DynamoDBStore) GetInventoryNodeByMAC(mac net.HardwareAddr) (*types.InventoryNode, error) {
	node, err := db.GetNodeByMAC(mac)
	if err != nil {
		return nil, err
	}

	return types.NewInventoryNode(node, db, db)
}

func (db *DynamoDBStore) GetNodeByID(id string) (*types.Node, error) {
	node := &types.Node{}
	err := db.getNewest(id, node)
	return node, err
}

func (db *DynamoDBStore) GetNodeByMAC(mac net.HardwareAddr) (*types.Node, error) {
	e := &NodeMacIndexEntry{}
	err := db.getNewest(mac.String(), e)
	if err != nil {
		return nil, err
	}

	return db.GetNodeByID(e.NodeID)
}

func (db *DynamoDBStore) GetNetworkByID(id string) (*types.Network, error) {
	network := &types.Network{}
	err := db.getNewest(id, network)
	if err != nil {
		return nil, err
	}

	if network.Subnets == nil {
		network.Subnets = make([]*types.Subnet, 0)
	}
	return network, err
}

func (db *DynamoDBStore) GetSystemByID(id string) (*types.System, error) {
	system := &types.System{}
	err := db.getNewest(id, system)
	return system, err
}

func (db *DynamoDBStore) UpdateFromInventoryStore(s RawInventoryStore) error {
	systems, err := s.GetSystems()
	if err != nil {
		return err
	}

	for _, system := range systems {
		err := db.Update(system)
		if err != nil {
			return err
		}
	}

	networks, err := s.GetNetworks()
	if err != nil {
		return err
	}

	for _, network := range networks {
		err := db.Update(network)
		if err != nil {
			return err
		}
	}

	nodes, err := s.GetNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		err := db.Update(node)
		if err != nil {
			return err
		}
	}

	return nil
}
