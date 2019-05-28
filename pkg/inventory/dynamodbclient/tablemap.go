package dynamodbclient

import (
	"reflect"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

// DynamoDBStoreTableMap maps data types to the appropriate table within DynamoDB
type DynamoDBStoreTableMap map[reflect.Type]DynamoDBStoreTable

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
		reflect.TypeOf(types.IPReservation{}): &IPReservationTable{Name: "inventory_ipam_ip"},
	}
)
