package dynamodbclient

import (
	"fmt"
	"net"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

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
