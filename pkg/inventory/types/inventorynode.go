package types

import (
	"fmt"
	"net"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/ipam"
)

type NetworkDB interface {
	GetNetworkByID(string) (*Network, error)
}

type SystemDB interface {
	GetSystemByID(string) (*System, error)
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
	Metadata        map[string]interface{}
	LastUpdated     time.Time
}

func NewInventoryNode(node *Node, networkDB NetworkDB, systemDB SystemDB) (*InventoryNode, error) {
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
	for netname, nicinfo := range node.Networks {
		network, err := networkDB.GetNetworkByID(netname)
		if err != nil {
			return nil, err
		}

		ips := make([]string, 0)
		gateways := make([]string, 0)
		dns := make([]string, 0)
		for _, subnet := range network.Subnets {
			if subnet.Cidr.Contains(nicinfo.IP) {
				ip := net.IPNet{IP: nicinfo.IP, Mask: subnet.Cidr.Mask}
				ips = append(ips, ip.String())
				for _, dnsIP := range subnet.DNS {
					dns = append(dns, dnsIP.String())
				}
				gateways = append(gateways, subnet.Gateway.String())
			} else if subnet.AllocationMethod == "static_inventory" {
				allocatedIp, err := ipam.GetIPByLocation(subnet.Cidr, node.ChassisLocation.Rack, node.ChassisLocation.BottomU, node.ChassisSubIndex)
				if err != nil {
					return nil, fmt.Errorf("error allocating ip from subnet: %v", err)
				}
				ip := net.IPNet{IP: allocatedIp, Mask: subnet.Cidr.Mask}
				ips = append(ips, ip.String())
				for _, dnsIP := range subnet.DNS {
					dns = append(dns, dnsIP.String())
				}
				gateways = append(gateways, subnet.Gateway.String())
			}
		}

		config := &NicConfig{
			IP:      ips,
			Gateway: gateways,
			DNS:     dns,
		}

		nicInstance := &NICInstance{NIC: *nicinfo, Network: *network, Config: *config}
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
	var ips []net.IP
	for _, nicinstance := range i.Networks {
		if nicinstance.NIC.IP != nil {
			ips = append(ips, nicinstance.NIC.IP)
		}
	}
	return ips
}
