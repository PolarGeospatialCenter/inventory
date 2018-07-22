package inventory

import (
	"fmt"
	"log"
	"net"
	"reflect"
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
type DynamoDBStoreTableMap map[reflect.Type]string

// LookupTable finds the table name associated with the type of the interface.
func (m DynamoDBStoreTableMap) LookupTable(t interface{}) string {
	var typ reflect.Type
	typ = reflect.TypeOf(t)

	// Check for direct match
	if table, ok := m[typ]; ok {
		return table
	}

	// Recurse to element/indirect type if no match is found, default to empty table name
	switch typ.Kind() {
	case reflect.Ptr:
		fallthrough
	case reflect.Array:
		fallthrough
	case reflect.Chan:
		fallthrough
	case reflect.Map:
		fallthrough
	case reflect.Slice:
		return m.LookupTable(reflect.Indirect(reflect.New(reflect.TypeOf(t).Elem())).Interface())
	default:
		return ""
	}
}

func (m DynamoDBStoreTableMap) Tables() []string {
	tables := make([]string, len(m))
	idx := 0
	for _, tablename := range m {
		tables[idx] = tablename
		idx++
	}
	return tables
}

var (
	defatultDynamoDBTables = &DynamoDBStoreTableMap{
		reflect.TypeOf(types.Node{}):        "inventory_nodes",
		reflect.TypeOf(types.Network{}):     "inventory_networks",
		reflect.TypeOf(types.System{}):      "inventory_systems",
		reflect.TypeOf(NodeMacIndexEntry{}): "inventory_node_mac_lookup",
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

func (i *NodeMacIndexEntry) SetTimestamp(timestamp time.Time) {
	i.LastUpdated = timestamp
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
func NewDynamoDBStore(db *dynamodb.DynamoDB, tableMap DynamoDBTableLookup) *DynamoDBStore {
	if tableMap == nil {
		tableMap = defatultDynamoDBTables
	}
	obj := &DynamoDBStore{tableMap: tableMap, db: db}
	return obj
}

func (db *DynamoDBStore) InitializeTables() error {
	for _, table := range db.tableMap.Tables() {
		_, err := db.db.DescribeTable(&dynamodb.DescribeTableInput{TableName: &table})
		if err == nil {
			continue
		}
		err = db.createTable(table)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DynamoDBStore) createTable(table string) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String("HASH"),
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

func (db *DynamoDBStore) Refresh() error {
	return nil
}

func (db *DynamoDBStore) Update(obj InventoryObject) error {
	// log.Printf("Updating %s: %d", obj.ID(), obj.Timestamp())
	putItem := &dynamodb.PutItemInput{}
	putItem.SetTableName(db.tableMap.LookupTable(obj))

	switch obj.(type) {
	case *types.Node:
		node := obj.(*types.Node)
		putItem.Item, _ = dynamodbattribute.MarshalMap(node)

		existingMacIndices, err := db.GetMacIndexEntriesByNodeID(node.ID())
		if err != nil {
			return fmt.Errorf("unable to lookup existing mac index entries: %v", err)
		}

		for _, nic := range node.Networks {
			if nic.MAC != nil {
				db.Update(&NodeMacIndexEntry{Mac: nic.MAC, LastUpdated: node.LastUpdated, NodeID: node.ID()})
				delete(existingMacIndices, nic.MAC.String())
			}
		}

		for _, oldMacIndex := range existingMacIndices {
			err := db.Delete(oldMacIndex)
			if err != nil {
				return fmt.Errorf("unable to delete previous mac index entry: %v", err)
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

	// log.Print(putItem.TableName)
	_, err := db.db.PutItem(putItem)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes the inventory object from dynamodb
func (db *DynamoDBStore) Delete(obj InventoryObject) error {
	objID, err := dynamodbattribute.Marshal(obj.ID())
	if err != nil {
		return fmt.Errorf("unable to marshal object id for deletion: %v", err)
	}

	table := db.tableMap.LookupTable(obj)
	partitionKey, err := db.getPartitionKey(table)
	if err != nil {
		return fmt.Errorf("unable to determine partition key for requested delete object type (%T): %v", obj, err)
	}
	deleteItem := &dynamodb.DeleteItemInput{}
	deleteItem.SetKey(map[string]*dynamodb.AttributeValue{partitionKey: objID})
	deleteItem.SetTableName(table)

	_, err = db.db.DeleteItem(deleteItem)
	return err
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
		ScanIndexForward:          aws.Bool(false),
		TableName:                 aws.String(table),
		KeyConditionExpression:    aws.String(queryString),
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

func (db *DynamoDBStore) getAll(out interface{}) error {
	table := db.tableMap.LookupTable(out)
	in := &dynamodb.ScanInput{
		TableName: aws.String(table),
	}

	outputElements := make([]map[string]*dynamodb.AttributeValue, 0, 0)
	scanFn := func(results *dynamodb.ScanOutput, lastPage bool) bool {
		for _, i := range results.Items {
			outputElements = append(outputElements, i)
		}
		return false
	}

	err := db.db.ScanPages(in, scanFn)
	if err != nil {
		return fmt.Errorf("unable to scan pages from dynamodb table %s: %v", table, err)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(outputElements, out)
	if err != nil {
		return err
	}

	return nil
}

func (db *DynamoDBStore) Exists(obj InventoryObject) (bool, error) {
	table := db.tableMap.LookupTable(obj)
	partitionKeyName, err := db.getPartitionKey(table)
	if err != nil {
		return false, err
	}

	queryValues, err := dynamodbattribute.MarshalMap(map[string]string{":partitionkeyval": obj.ID()})
	if err != nil {
		return false, err
	}

	queryString := fmt.Sprintf("%s=:partitionkeyval", partitionKeyName)
	q := &dynamodb.QueryInput{
		ScanIndexForward:          aws.Bool(false),
		TableName:                 aws.String(table),
		KeyConditionExpression:    aws.String(queryString),
		ExpressionAttributeValues: queryValues,
	}

	results, err := db.db.Query(q)
	if err != nil {
		return false, err
	}

	return len(results.Items) != 0, nil
}

func (db *DynamoDBStore) GetByID(id string, obj InventoryObject) error {
	err := db.getNewest(id, obj)
	if _, ok := err.(ErrDynamoDBRecordNotFound); ok {
		return ErrObjectNotFound
	}
	return err
}

func (db *DynamoDBStore) GetInventoryNodes() (map[string]*types.InventoryNode, error) {
	nodes, err := db.GetNodes()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup nodes: %v", err)
	}

	networks, err := db.GetNetworks()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup networks: %v", err)
	}

	systems, err := db.GetSystems()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup systems: %v", err)
	}

	out := make(map[string]*types.InventoryNode)
	for _, n := range nodes {
		iNode, err := types.NewInventoryNode(n, types.NetworkMap(networks), types.SystemMap(systems))
		if err != nil {
			return nil, fmt.Errorf("unable to compile inventory node: %v", err)
		}
		out[n.ID()] = iNode
	}
	return out, nil
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

func (db *DynamoDBStore) GetNodes() (map[string]*types.Node, error) {
	nodeList := make([]*types.Node, 0, 0)
	err := db.getAll(&nodeList)
	if err != nil {
		return nil, fmt.Errorf("error getting all nodes: %v", err)
	}
	nodes := make(map[string]*types.Node)
	for _, n := range nodeList {
		nodes[n.ID()] = n
	}
	return nodes, nil
}

func (db *DynamoDBStore) GetNodeByID(id string) (*types.Node, error) {
	node := &types.Node{}
	err := db.GetByID(id, node)
	return node, err
}

func (db *DynamoDBStore) GetNodeByMAC(mac net.HardwareAddr) (*types.Node, error) {
	e := &NodeMacIndexEntry{}
	err := db.getNewest(mac.String(), e)
	if _, ok := err.(ErrDynamoDBRecordNotFound); ok {
		return nil, ErrObjectNotFound
	} else if err != nil {
		return nil, err
	}

	return db.GetNodeByID(e.NodeID)
}

func (db *DynamoDBStore) GetNetworks() (map[string]*types.Network, error) {
	networkList := make([]*types.Network, 0, 0)
	err := db.getAll(&networkList)
	if err != nil {
		return nil, fmt.Errorf("error getting all networks: %v", err)
	}
	log.Printf("Network List returned: %v", networkList)
	networks := make(map[string]*types.Network)
	for _, n := range networkList {
		networks[n.ID()] = n
	}
	return networks, nil
}

func (db *DynamoDBStore) GetNetworkByID(id string) (*types.Network, error) {
	network := &types.Network{}
	err := db.GetByID(id, network)
	if err != nil {
		return nil, err
	}

	if network.Subnets == nil {
		network.Subnets = make([]*types.Subnet, 0)
	}
	return network, err
}

func (db *DynamoDBStore) GetSystems() (map[string]*types.System, error) {
	systemList := make([]*types.System, 0, 0)
	err := db.getAll(&systemList)
	if err != nil {
		return nil, fmt.Errorf("error getting all systems: %v", err)
	}
	systems := make(map[string]*types.System)
	for _, s := range systemList {
		systems[s.ID()] = s
	}
	return systems, nil
}

func (db *DynamoDBStore) GetSystemByID(id string) (*types.System, error) {
	system := &types.System{}
	err := db.GetByID(id, system)
	return system, err
}

func (db *DynamoDBStore) GetMacIndexEntriesByNodeID(id string) (map[string]*NodeMacIndexEntry, error) {
	allMacs := make([]*NodeMacIndexEntry, 0, 0)
	err := db.getAll(&allMacs)
	if err != nil {
		return nil, fmt.Errorf("unable to get all NodeMacIndexEntries: %v", err)
	}

	results := make(map[string]*NodeMacIndexEntry, 0)
	for _, nodeMacIndexEntry := range allMacs {
		if nodeMacIndexEntry.NodeID == id {
			results[nodeMacIndexEntry.Mac.String()] = nodeMacIndexEntry
		}
	}
	return results, nil
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
