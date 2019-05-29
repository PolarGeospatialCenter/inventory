package dynamodbclient

import (
	"fmt"
	"net"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/inventory/pkg/ipam"
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
	for netname, nic := range newNode.Networks {
		network, err := db.Network().GetNetworkByID(netname)
		if err != nil {
			return err
		}

		if nic.IP == nil && nic.MAC == nil {
			continue
		}

		if nic.MAC != nil {
			db.nodeMacIndex().Create(&NodeMacIndexEntry{Mac: nic.MAC, LastUpdated: newNode.LastUpdated, NodeID: newNode.ID()})
		}

		reservation, err := generateIPReservation(newNode, network)
		if err != nil {
			return fmt.Errorf("unexpected error while creating reservation for node '%s' on network '%s': %v", newNode.InventoryID, network.ID(), err)
		}

		if reservation == nil {
			continue
		}

		reservation.HostInformation = newNode.InventoryID
		err = db.IPReservation().CreateOrUpdateIPReservation(reservation)
		if err != nil {
			return fmt.Errorf("unable to reserve IP for NIC: %v", err)
		}
		nic.IP = reservation.IP.IP

	}

	return db.DynamoDBStore.create(newNode)
}

func (db *NodeStore) Update(updatedNode *types.Node) error {
	existingNode := *updatedNode
	err := db.DynamoDBStore.get(&existingNode)
	if err != nil {
		return err
	}

	existingMacIndices, err := db.nodeMacIndex().GetMacIndexEntriesByNodeID(updatedNode.ID())
	if err != nil {
		return fmt.Errorf("unable to lookup existing mac index entries: %v", err)
	}

	for _, oldMacIndex := range existingMacIndices {
		err := db.nodeMacIndex().Delete(oldMacIndex)
		if err != nil {
			return fmt.Errorf("unable to delete previous mac index entry: %v", err)
		}
	}

	for netname, nic := range existingNode.Networks {
		if updatedNic, ok := updatedNode.Networks[netname]; ok &&
			updatedNic.IP.String() == nic.IP.String() &&
			updatedNic.MAC.String() == nic.MAC.String() {
			continue
		}

		network, err := db.Network().GetNetworkByID(netname)
		if err != nil {
			return fmt.Errorf("unable to get network named '%s': %v", netname, err)
		}

		deleteSubnet := network.GetSubnetContainingIP(nic.IP)
		if deleteSubnet == nil {
			continue
		}
		err = db.IPReservation().Delete(&types.IPReservation{IP: &net.IPNet{IP: nic.IP, Mask: deleteSubnet.Cidr.Mask}})
		if err != nil {
			return fmt.Errorf("unable to delete IP for NIC: %v", err)
		}
	}

	for netname, nic := range updatedNode.Networks {
		if nic.MAC != nil {
			db.nodeMacIndex().Create(&NodeMacIndexEntry{Mac: nic.MAC, LastUpdated: updatedNode.LastUpdated, NodeID: updatedNode.ID()})
			delete(existingMacIndices, nic.MAC.String())
		}

		if existingNic, ok := existingNode.Networks[netname]; ok &&
			existingNic.IP.String() == nic.IP.String() &&
			existingNic.MAC.String() == nic.MAC.String() {
			continue
		}

		network, err := db.Network().GetNetworkByID(netname)
		if err != nil {
			return fmt.Errorf("error getting networks: %v", err)
		}

		reservation, err := generateIPReservation(updatedNode, network)
		if err != nil {
			return fmt.Errorf("unexpected error while creating reservation for node '%s' on network '%s': %v", updatedNode.InventoryID, network.ID(), err)
		}

		if reservation == nil {
			continue
		}

		err = db.IPReservation().CreateOrUpdateIPReservation(reservation)
		if err != nil {
			return fmt.Errorf("unable to reserve IP for NIC: %v", err)
		}
		nic.IP = reservation.IP.IP
	}
	return db.DynamoDBStore.update(updatedNode)
}

func generateIPReservation(node *types.Node, network *types.Network) (*types.IPReservation, error) {
	var ip *net.IPNet
	nic := node.Networks[network.ID()]

	for _, subnet := range network.Subnets {
		if nic.IP == nil {
			allocatedIP, _, _, err := subnet.GetNicConfig(node)
			if err == nil {
				ip = &allocatedIP
				break
			} else if err != ipam.ErrAllocationNotImplemented {
				return nil, fmt.Errorf("unexpected error allocating IP for nic: %v", err)
			}
		} else if subnet.Cidr.Contains(nic.IP) {
			ip = &net.IPNet{
				IP:   nic.IP,
				Mask: subnet.Cidr.Mask,
			}
		}
	}

	if ip == nil {
		return nil, nil
	}

	sTime := time.Now()
	reservation := &types.IPReservation{IP: ip, MAC: nic.MAC, Start: &sTime}
	return reservation, nil
}

func (db *NodeStore) Exists(node *types.Node) (bool, error) {
	return db.DynamoDBStore.exists(node)
}

func (db *NodeStore) Delete(node *types.Node) error {
	for netname, nic := range node.Networks {
		if nic.MAC != nil {
			err := db.nodeMacIndex().Delete(&NodeMacIndexEntry{Mac: nic.MAC})
			if err != nil {
				return fmt.Errorf("error removing mac index entry (%s) for this node: %v", nic.MAC.String(), err)
			}
		}

		if nic.IP != nil {
			network, err := db.Network().GetNetworkByID(netname)
			if err != nil {
				return fmt.Errorf("error getting networks: %v", err)
			}

			reservation, err := generateIPReservation(node, network)
			if err != nil {
				return fmt.Errorf("unexpected error while creating reservation for node '%s' on network '%s': %v", node.InventoryID, network.ID(), err)
			}

			err = db.IPReservation().Delete(reservation)
			if err != nil {
				return fmt.Errorf("error deleting reservation for ip '%s': %v", nic.IP.String(), err)
			}
		}
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
