package dynamodbclient

import (
	"fmt"
	"net"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

type InventoryNodeStore struct {
	*DynamoDBStore
}

func (db *InventoryNodeStore) GetInventoryNodes() (map[string]*types.InventoryNode, error) {
	nodes, err := db.Node().GetNodes()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup nodes: %v", err)
	}

	networks, err := db.Network().GetNetworks()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup networks: %v", err)
	}

	systems, err := db.System().GetSystems()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup systems: %v", err)
	}

	ipResrvations, err := db.IPReservation().GetAllIPReservations()
	if err != nil {
		return nil, fmt.Errorf("unable to get all ip reservations: %v", err)
	}

	ipReservationMap := make(types.IPReservationMap, len(ipResrvations))
	for _, r := range ipResrvations {
		ipReservationMap.Add(r)
	}

	out := make(map[string]*types.InventoryNode)
	for _, n := range nodes {
		iNode, err := types.NewInventoryNode(n, types.NetworkMap(networks), types.SystemMap(systems), ipReservationMap)
		if err != nil {
			return nil, fmt.Errorf("unable to compile inventory node: %v", err)
		}
		out[n.ID()] = iNode
	}
	return out, nil
}

func (db *InventoryNodeStore) GetInventoryNodeByID(id string) (*types.InventoryNode, error) {
	node, err := db.Node().GetNodeByID(id)
	if err != nil {
		return nil, err
	}

	return types.NewInventoryNode(node, db.Network(), db.System(), db.IPReservation())
}

func (db *InventoryNodeStore) GetInventoryNodeByMAC(mac net.HardwareAddr) (*types.InventoryNode, error) {
	node, err := db.Node().GetNodeByMAC(mac)
	if err != nil {
		return nil, err
	}

	return types.NewInventoryNode(node, db.Network(), db.System(), db.IPReservation())
}
