package dynamodbclient

import (
	"fmt"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type DynamoDBTableLookup interface {
	LookupTable(interface{}) DynamoDBStoreTable
	Tables() []DynamoDBStoreTable
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
