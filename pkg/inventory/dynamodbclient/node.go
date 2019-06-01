package dynamodbclient

import (
	"fmt"
	"net"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

type NodeStore struct {
	*DynamoDBStore
}

func (db *NodeStore) GetNodes() (map[string]*types.Node, error) {
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

func (db *NodeStore) GetNodeByID(id string) (*types.Node, error) {
	node := &types.Node{}
	node.InventoryID = id
	err := db.get(node)
	return node, err
}

func (db *NodeStore) GetNodeByMAC(mac net.HardwareAddr) (*types.Node, error) {
	e := &NodeMacIndexEntry{}
	e.Mac = mac
	err := db.get(e)
	if err != nil {
		return nil, err
	}

	return db.GetNodeByID(e.NodeID)
}

func (db *NodeStore) Create(newNode *types.Node) error {

	err := db.reconcileIPs(newNode)
	if err != nil {
		return err
	}

	err = db.reconcileMacIndex(newNode)
	if err != nil {
		return nil
	}

	return db.DynamoDBStore.create(newNode)
}

func (db *NodeStore) reconcileIPs(node *types.Node) error {
	// For a given node, make sure that the list of IPs on each interface is valid and fully populated
	existingNode, err := db.GetNodeByID(node.ID())
	if err != nil && err != ErrObjectNotFound {
		return fmt.Errorf("a node with this id exists already, but we can't get it for comparison: %v", err)
	}

	macsToRemove := make(map[string]net.HardwareAddr)

	if existingNode != nil {
		for _, iface := range existingNode.Networks {
			for _, mac := range iface.NICs {
				macsToRemove[mac.String()] = mac
			}
		}
	}

	for netname, iface := range node.Networks {
		// TODO: do we support static IPs without macs?
		if iface.NICs == nil || len(iface.NICs) == 0 {
			// if we don't have a mac, we can't reserve any IPs
			continue
		}

		// Get network
		network, err := db.Network().GetNetworkByID(netname)
		if err != nil {
			return fmt.Errorf("unable to get network named '%s': %v", netname, err)
		}

		// For each subnet with an allocation strategy, make sure theres a valid static IP reservation
		for _, subnet := range network.Subnets {
			for _, mac := range iface.NICs {
				// this NIC still exists, we don't need to remove
				if _, ok := macsToRemove[mac.String()]; ok {
					delete(macsToRemove, mac.String())
				}
			}

			if !subnet.StaticAllocationEnabled() {
				continue
			}

			existingReservations := types.IPReservationList{}
			for _, mac := range iface.NICs {
				reservation, err := db.IPReservation().GetExistingIPReservationInSubnet(subnet.Cidr, mac)
				if err != nil {
					return fmt.Errorf("unable to get reservation for nic: %v", err)
				}
				if reservation != nil {
					existingReservations = append(existingReservations, reservation)
				}
			}
			// If allocation is enabled for this subnet, then we should have one static reservation.
			// If allocation is disabled, a static reservation may exist, but we will not create one.
			// Check for existing reservations
			// If static reservation exists for any mac on this iface, continue to next interface
			// If we find a valid dynamic reservation for any mac on the iface make it static
			// otherwise have the allocator choose an address and create a static reservation
			existingStatic := existingReservations.Static().ValidAt(time.Now())
			if len(existingStatic) > 0 {
				continue
			}

			// try to upgrade a dynamic reservation
			existingDynamic := existingReservations.Dynamic().ValidAt(time.Now())
			if len(existingDynamic) > 0 {
				reservation := existingDynamic[0]
				reservation.End = nil
				err = db.IPReservation().UpdateIPReservation(reservation)
				if err != nil {
					return fmt.Errorf("unable to upgrade dynamic reservation: %v", err)
				}
				continue
			}

			// we don't have a static or dynamic reservation, and allocation is enabled
			// allocate an IP and create a reservation
			_, err := db.IPReservation().CreateRandomIPReservation(types.NewStaticIPReservation(), subnet)
			if err != nil {
				return err
			}
		}
	}

	for _, mac := range macsToRemove {
		reservations, err := db.IPReservation().GetIPReservationsByMac(mac)
		for _, reservation := range reservations.Static() {
			err = db.IPReservation().Delete(reservation)
			if err != nil {
				return fmt.Errorf("unable to delete reservation: %v", err)
			}
		}
	}
	return nil
}

func (db *NodeStore) reconcileMacIndex(node *types.Node) error {
	existingMacIndices, err := db.nodeMacIndex().GetMacIndexEntriesByNodeID(node.ID())
	if err != nil {
		return fmt.Errorf("unable to lookup existing mac index entries: %v", err)
	}

	newMacs := make(map[string]net.HardwareAddr)

	for _, iface := range node.Networks {
		for _, mac := range iface.NICs {
			newMacs[mac.String()] = mac
		}
	}

	for _, oldMacIndex := range existingMacIndices {
		if _, ok := newMacs[oldMacIndex.Mac.String()]; ok {
			delete(newMacs, oldMacIndex.Mac.String())
			continue
		}
		err := db.nodeMacIndex().Delete(oldMacIndex)
		if err != nil {
			return fmt.Errorf("unable to delete previous mac index entry: %v", err)
		}
	}

	for _, mac := range newMacs {
		err = db.nodeMacIndex().Create(&NodeMacIndexEntry{Mac: mac, LastUpdated: node.LastUpdated, NodeID: node.ID()})
		if err != nil {
			return fmt.Errorf("unable to create mac index entry: %v", err)
		}
	}
	return nil
}

func (db *NodeStore) Update(updatedNode *types.Node) error {
	err := db.reconcileIPs(updatedNode)
	if err != nil {
		return err
	}

	err = db.reconcileMacIndex(updatedNode)
	if err != nil {
		return nil
	}

	return db.DynamoDBStore.update(updatedNode)
}

func (db *NodeStore) Exists(node *types.Node) (bool, error) {
	return db.DynamoDBStore.exists(node)
}

func (db *NodeStore) Delete(node *types.Node) error {
	node.Networks = types.NICInfoMap{}
	err := db.reconcileIPs(node)
	if err != nil {
		return err
	}

	err = db.reconcileMacIndex(node)
	if err != nil {
		return nil
	}
	return db.DynamoDBStore.delete(node)
}

func (db *NodeStore) ObjDelete(obj interface{}) error {
	node, ok := obj.(*types.Node)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Delete(node)
}

func (db *NodeStore) ObjCreate(obj interface{}) error {
	node, ok := obj.(*types.Node)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Create(node)
}

func (db *NodeStore) ObjUpdate(obj interface{}) error {
	node, ok := obj.(*types.Node)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Update(node)
}

func (db *NodeStore) ObjExists(obj interface{}) (bool, error) {
	node, ok := obj.(*types.Node)
	if !ok {
		return false, ErrInvalidObjectType
	}
	return db.Exists(node)
}
