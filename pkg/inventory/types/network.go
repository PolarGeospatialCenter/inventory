package types

import "time"

type Network struct {
	Name        string
	MTU         uint
	Subnets     []*Subnet
	Domain      string
	Metadata    map[string]interface{}
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
