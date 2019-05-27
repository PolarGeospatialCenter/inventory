package types

import (
	"net"
	"time"
)

type Network struct {
	Name        string
	MTU         uint
	Subnets     SubnetList
	Domain      string
	Metadata    Metadata
	LastUpdated time.Time
}

func NewNetwork() *Network {
	return &Network{}
}

func (n *Network) ID() string {
	return n.Name
}

func (n *Network) Timestamp() int64 {
	return n.LastUpdated.Unix()
}

func (n *Network) SetTimestamp(timestamp time.Time) {
	n.LastUpdated = timestamp
}

func (n *Network) GetSubnetContainingIP(ip net.IP) *Subnet {
	for _, subnet := range n.Subnets {
		if subnet.Cidr.Contains(ip) {
			return subnet
		}
	}

	return nil
}
