package types

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/PolarGeospatialCenter/inventory/pkg/ipam"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type SubnetList []*Subnet

// Subnet stores information about an IP subnet
type Subnet struct {
	Name             string
	Cidr             *net.IPNet
	Gateway          net.IP
	DNS              []net.IP
	AllocationMethod string
}

// MarshalJSON implements the Marshaler Interface so that cidr is rendered as a
// string.
func (s *Subnet) MarshalJSON() ([]byte, error) {
	type Alias Subnet
	v := &struct {
		*Alias
		Cidr string
	}{
		Alias: (*Alias)(s),
	}
	v.Cidr = s.Cidr.String()
	return json.Marshal(v)
}

// UnmarshalJSON implements Unmarshaler interface so that cidr can be directly
// read from a string
func (s *Subnet) UnmarshalJSON(data []byte) error {
	type Alias Subnet
	v := &struct {
		*Alias
		Cidr string
	}{
		Alias: (*Alias)(s),
	}
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	_, cidr, err := net.ParseCIDR(v.Cidr)
	s.Cidr = cidr
	return err
}

// GetNicConfig Allocates IP address for the specified node, and returns dns gateway info if applicable (or no particular node if the allocator supports it)
func (s *Subnet) GetNicConfig(node *Node) (net.IPNet, []net.IP, net.IP, error) {
	switch s.AllocationMethod {
	case "static_inventory":
		id, err := node.NumericId()
		if err != nil {
			return net.IPNet{}, []net.IP{}, net.IP{}, fmt.Errorf("error getting numeric node id: %v", err)
		}
		allocatedIp, err := ipam.GetIpById(id, s.Cidr, s.Gateway)
		if err != nil {
			return net.IPNet{}, []net.IP{}, net.IP{}, fmt.Errorf("error allocating ip from subnet: %v", err)
		}
		ip := net.IPNet{IP: allocatedIp, Mask: s.Cidr.Mask}
		return ip, s.DNS, s.Gateway, nil
	case "static_location":
		allocatedIp, err := ipam.GetIPByLocation(s.Cidr, node.ChassisLocation.Rack, node.ChassisLocation.BottomU, node.ChassisSubIndex)
		if err != nil {
			return net.IPNet{}, []net.IP{}, net.IP{}, fmt.Errorf("error allocating ip from subnet: %v", err)
		}
		ip := net.IPNet{IP: allocatedIp, Mask: s.Cidr.Mask}
		return ip, s.DNS, s.Gateway, nil
	}

	return net.IPNet{}, []net.IP{}, net.IP{}, ipam.ErrAllocationNotImplemented
}

// UnmarshalYAML unmarshals a NICInfo object, converting cidr from a string
func (s *Subnet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := &struct {
		Name             string
		Gateway          net.IP
		Cidr             string
		AllocationMethod string
		DNS              []net.IP
	}{}
	err := unmarshal(v)
	if err != nil {
		return err
	}
	_, cidr, err := net.ParseCIDR(v.Cidr)
	s.Name = v.Name
	s.Gateway = v.Gateway
	s.DNS = v.DNS
	s.AllocationMethod = v.AllocationMethod
	s.Cidr = cidr
	return err
}

func (n *SubnetList) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	if av.L != nil {
		l := make(SubnetList, 0, len(av.L))
		for _, item := range av.L {
			subnet := &Subnet{}
			err := dynamodbattribute.Unmarshal(item, subnet)
			if err != nil {
				return err
			}
			l = append(l, subnet)
		}
		*n = l
		return nil
	} else if av.NULL != nil && *av.NULL {
		*n = SubnetList{}
	}
	return nil
}
