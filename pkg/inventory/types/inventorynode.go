package types

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/azenk/iputils"
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
		for _, subnet := range network.Subnets {
			if subnet.Cidr.Contains(nicinfo.IP) {
				ip := net.IPNet{IP: nicinfo.IP, Mask: subnet.Cidr.Mask}
				ips = append(ips, ip.String())
				gateways = append(gateways, subnet.Gateway.String())
			}
		}

		config := &NicConfig{
			IP:      ips,
			Gateway: gateways,
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

// GetNodeAllocation returns a unique ipv6 allocation for a node
func (i *InventoryNode) GetNodeAllocation(logicalNetworkName string, subnetID int) (string, error) {
	nic, ok := i.Networks[logicalNetworkName]
	if !ok {
		return "", fmt.Errorf("requested network does not exist: %s", logicalNetworkName)
	}

	if subnetID >= len(nic.Network.Subnets) {
		return "", fmt.Errorf("requested subnet out of bounds")
	}

	subnetCidr := nic.Network.Subnets[subnetID].Cidr
	if subnetCidr.IP.To4() != nil {
		return "", fmt.Errorf("getting an allocation from a v4 subnet isn't supported")
	}

	var serialNumber uint64
	serialNumberWidth := 16

	re := regexp.MustCompile("[^0-9]*([0-9]*)")
	matches := re.FindStringSubmatch(i.ID())
	serialNumber, err := strconv.ParseUint(matches[1], 10, serialNumberWidth)
	if err != nil {
		return "", fmt.Errorf("unable to parse serialnumber from inventory ID: %v", err)
	}

	if serialNumber >= (1 << uint(serialNumberWidth)) {
		return "", fmt.Errorf("node serial number too large")
	}

	startoffset, _ := subnetCidr.Mask.Size()
	newIP, err := iputils.SetBits(subnetCidr.IP, serialNumber, uint(startoffset), uint(serialNumberWidth))

	return newIP.String(), err
}
