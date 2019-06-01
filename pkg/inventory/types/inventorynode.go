package types

import (
	"fmt"
	"net"
	"time"
)

type NetworkDB interface {
	GetNetworkByID(string) (*Network, error)
}

type SystemDB interface {
	GetSystemByID(string) (*System, error)
}

type IPReservationDB interface {
	GetIPReservationsByMac(net.HardwareAddr) (IPReservationList, error)
}

type NetworkMap map[string]*Network

func (m NetworkMap) GetNetworkByID(id string) (*Network, error) {
	network, ok := m[id]
	if !ok {
		return nil, fmt.Errorf("Network not found matching id: %s", id)
	}
	return network, nil
}

type SystemMap map[string]*System

func (m SystemMap) GetSystemByID(id string) (*System, error) {
	system, ok := m[id]
	if !ok {
		return nil, fmt.Errorf("System not found matching id: %s", id)
	}
	return system, nil
}

type IPReservationMap map[string]IPReservationList

func (m IPReservationMap) GetIPReservationsByMac(mac net.HardwareAddr) (IPReservationList, error) {
	reservations, ok := m[mac.String()]
	if !ok {
		return nil, fmt.Errorf("IPReservations not found matching mac: %s", mac.String())
	}
	return reservations, nil
}

type InventoryNode struct {
	Hostname        string
	LocationString  string
	InventoryID     string
	Tags            []string
	Networks        map[string]*NICInstance
	Role            string
	Location        *ChassisLocation
	ChassisSubIndex string
	System          *System
	Environment     *Environment
	Metadata        Metadata
	LastUpdated     time.Time
	ips             []net.IP
}

func NewInventoryNode(node *Node, networkDB NetworkDB, systemDB SystemDB, ipReservationDB IPReservationDB) (*InventoryNode, error) {
	inode := &InventoryNode{}
	inode.Hostname = node.Hostname()
	inode.LocationString = node.Location()
	inode.InventoryID = node.InventoryID
	inode.Tags = node.Tags
	inode.Location = node.ChassisLocation
	if node.Metadata != nil {
		inode.Metadata = node.Metadata
	}
	inode.ChassisSubIndex = node.ChassisSubIndex

	lastUpdate := node.LastUpdated

	system, err := systemDB.GetSystemByID(node.System)
	if err != nil {
		return nil, err
	}
	inode.System = system
	if inode.System.LastUpdated.Unix() > lastUpdate.Unix() {
		lastUpdate = inode.System.LastUpdated
	}

	for _, role := range inode.System.Roles {
		if role == node.Role {
			inode.Role = role
			break
		}
	}
	if inode.Role == "" {
		return nil, fmt.Errorf("No system role found matching: %s", node.Role)
	}

	environment, ok := inode.System.Environments[node.Environment]
	if !ok {
		return nil, fmt.Errorf("No environment found matching %s", node.Environment)
	}
	inode.Environment = environment

	inode.Networks = make(map[string]*NICInstance)
	for netname, iface := range node.Networks {
		network, err := networkDB.GetNetworkByID(netname)
		if err != nil {
			return nil, err
		}

		reservations := IPReservationList{}
		for _, mac := range iface.NICs {
			ifaceReservations, err := ipReservationDB.GetIPReservationsByMac(mac)
			if err != nil {
				return nil, err
			}
			reservations = append(reservations, ifaceReservations...)
		}
		for _, r := range reservations {
			inode.ips = append(inode.ips, r.IP.IP)
		}
		config := network.GetNicConfig(reservations)

		nicInstance := &NICInstance{Interface: *iface, Network: *network, Config: *config}
		logical, err := inode.Environment.LookupLogicalNetworkName(netname)
		if err != nil {
			return nil, err
		}
		inode.Networks[logical] = nicInstance
		if nicInstance.Network.LastUpdated.Unix() > lastUpdate.Unix() {
			lastUpdate = nicInstance.Network.LastUpdated
		}
	}

	inode.LastUpdated = lastUpdate
	return inode, nil
}

func (i *InventoryNode) ID() string {
	if i.InventoryID != "" {
		return i.InventoryID
	} else {
		return i.Hostname
	}
}

func (i *InventoryNode) Timestamp() int64 {
	return i.LastUpdated.Unix()
}

// IPs returns a slice containing all non-nil IPs assigned to the node
func (i *InventoryNode) IPs() []net.IP {
	return i.ips
}
