package types

import "time"

type Network struct {
	Name        string
	MTU         uint
	Subnets     []*Subnet
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
