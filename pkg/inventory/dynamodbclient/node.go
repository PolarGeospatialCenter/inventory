package dynamodbclient

import (
	"fmt"
	"net"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

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
	if err != nil {
		return nil, err
	}

	return db.GetNodeByID(e.NodeID)
}
