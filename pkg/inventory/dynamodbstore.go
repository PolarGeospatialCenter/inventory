package inventory

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/dynamodbstore"
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
type DynamoDBStoreTableMap map[reflect.Type]DynamoDBStoreTable

// DynamoDBStoreTable defines an interface for describing a dynamodb table
type DynamoDBStoreTable interface {
	GetName() string
	GetKeySchema() []*dynamodb.KeySchemaElement
	GetPartitionKeyName() string
	GetKeyAttributeDefinitions() []*dynamodb.AttributeDefinition
	GetKeyFrom(interface{}) (map[string]*dynamodb.AttributeValue, error)
	GetItemQueryInputFrom(interface{}) (*dynamodb.QueryInput, error)
}

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

// LookupTable finds the table name associated with the type of the interface.
func (m DynamoDBStoreTableMap) LookupTable(t interface{}) DynamoDBStoreTable {
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
		return nil
	}
}

func (m DynamoDBStoreTableMap) Tables() []DynamoDBStoreTable {
	tables := make([]DynamoDBStoreTable, len(m))
	idx := 0
	for _, table := range m {
		tables[idx] = table
		idx++
	}
	return tables
}

var (
	defatultDynamoDBTables = &DynamoDBStoreTableMap{
		reflect.TypeOf(types.Node{}):          &SimpleDynamoDBInventoryTable{Name: "inventory_nodes"},
		reflect.TypeOf(types.Network{}):       &SimpleDynamoDBInventoryTable{Name: "inventory_networks"},
		reflect.TypeOf(types.System{}):        &SimpleDynamoDBInventoryTable{Name: "inventory_systems"},
		reflect.TypeOf(NodeMacIndexEntry{}):   &SimpleDynamoDBInventoryTable{Name: "inventory_node_mac_lookup"},
		reflect.TypeOf(types.IPReservation{}): &dynamodbstore.IPReservationTable{Name: "inventory_ipam_ip"},
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
	LookupTable(interface{}) DynamoDBStoreTable
	Tables() []DynamoDBStoreTable
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
		if table == nil {
			continue
		}
		_, err := db.db.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(table.GetName())})
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

func (db *DynamoDBStore) createTable(table DynamoDBStoreTable) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: table.GetKeyAttributeDefinitions(),
		KeySchema:            table.GetKeySchema(),
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		TableName: aws.String(table.GetName()),
	}

	_, err := db.db.CreateTable(input)
	return err
}

func (db *DynamoDBStore) Refresh() error {
	return nil
}

func (db *DynamoDBStore) Update(obj interface{}) error {
	// log.Printf("Updating %s: %d", obj.ID(), obj.Timestamp())
	table := db.tableMap.LookupTable(obj)
	if table == nil {
		return fmt.Errorf("No table found for object of type %T", obj)
	}
	putItem := &dynamodb.PutItemInput{}
	putItem.SetTableName(table.GetName())

	switch o := obj.(type) {
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
	default:
		putItem.Item, _ = dynamodbattribute.MarshalMap(o)
	}

	keyMap, err := table.GetKeyFrom(obj)
	if err != nil {
		return err
	}

	for k, v := range keyMap {
		putItem.Item[k] = v
	}

	// log.Print(putItem.TableName)
	_, err = db.db.PutItem(putItem)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes the inventory object from dynamodb
func (db *DynamoDBStore) Delete(obj interface{}) error {
	table := db.tableMap.LookupTable(obj)
	if table == nil {
		return fmt.Errorf("No table found for object of type %T", obj)
	}

	deleteItem := &dynamodb.DeleteItemInput{}
	objKey, err := table.GetKeyFrom(obj)
	if err != nil {
		return fmt.Errorf("unable to get key from object: %v", err)
	}
	deleteItem.SetKey(objKey)
	deleteItem.SetTableName(table.GetName())

	_, err = db.db.DeleteItem(deleteItem)
	return err
}

func (db *DynamoDBStore) getAll(out interface{}) error {
	table := db.tableMap.LookupTable(out)
	if table == nil {
		return fmt.Errorf("No table found for object of type %T", out)
	}
	in := &dynamodb.ScanInput{
		TableName: aws.String(table.GetName()),
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

func (db *DynamoDBStore) Get(obj interface{}) error {
	table := db.tableMap.LookupTable(obj)
	if table == nil {
		return fmt.Errorf("No table found for object of type %T", obj)
	}

	q, err := table.GetItemQueryInputFrom(obj)
	if err != nil {
		return err
	}

	q.ScanIndexForward = aws.Bool(false)

	results, err := db.db.Query(q)
	if err != nil {
		return err
	}

	if len(results.Items) == 0 {
		return ErrObjectNotFound
	}

	if len(results.Items) > 1 {
		return fmt.Errorf("unable to lookup exactly one item: found %d matching", len(results.Items))
	}

	err = dynamodbattribute.UnmarshalMap(results.Items[0], obj)
	if err != nil {
		return err
	}
	return nil
}

func (db *DynamoDBStore) Exists(obj interface{}) (bool, error) {
	table := db.tableMap.LookupTable(obj)
	if table == nil {
		return false, fmt.Errorf("No table found for object of type %T", obj)
	}

	q, err := table.GetItemQueryInputFrom(obj)
	if err != nil {
		return false, err
	}

	q.ScanIndexForward = aws.Bool(false)

	results, err := db.db.Query(q)
	if err != nil {
		return false, err
	}

	return len(results.Items) != 0, nil
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
	node.InventoryID = id
	err := db.Get(node)
	return node, err
}

func (db *DynamoDBStore) GetNodeByMAC(mac net.HardwareAddr) (*types.Node, error) {
	e := &NodeMacIndexEntry{}
	e.Mac = mac
	err := db.Get(e)
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
	network.Name = id
	err := db.Get(network)
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
	system.Name = id
	err := db.Get(system)
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
