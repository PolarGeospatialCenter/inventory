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

// GetNicConfig builds a NicConfig object fo the specified interface on this network
func (n *Network) GetNicConfig(reservations IPReservationList) *NicConfig {
	nicConfig := NewNicConfig()
	for _, s := range n.Subnets {
		for _, r := range reservations {
			if !s.Cidr.Contains(r.IP.IP) {
				continue
			}
			nicConfig.IP = append(nicConfig.IP, r.IP.String())
			if s.Gateway != nil {
				nicConfig.Gateway = append(nicConfig.Gateway, s.Gateway.String())
			}
			for _, dns := range s.DNS {
				nicConfig.DNS = append(nicConfig.DNS, dns.String())
			}
		}
	}
	return nicConfig
}
