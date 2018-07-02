package types

import (
	"encoding/json"
	"net"

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
