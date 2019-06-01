package types

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// stringNICInfo is used for marshal and unmarshal wrappers
type stringNICInfo struct {
	MAC string
	IP  string
}

func (s *stringNICInfo) getMAC() (net.HardwareAddr, error) {
	if s.MAC != "" {
		return net.ParseMAC(s.MAC)
	}
	return net.HardwareAddr{}, nil
}

func (s *stringNICInfo) getIP() (net.IP, error) {
	return net.ParseIP(s.IP), nil
}

func (s *stringNICInfo) populateNICInfo(n *NICInfo) error {
	mac, err := s.getMAC()
	if err != nil {
		return err
	}
	n.MAC = mac

	ip, err := s.getIP()
	if err != nil {
		return err
	}
	n.IP = ip
	return nil
}

type NICInfoMap map[string]*NetworkInterface

// NICInfo describes a network interface
type NICInfo struct {
	MAC net.HardwareAddr
	IP  net.IP
}

// MarshalJSON marshals a NICInfo object, converting IP and MAC to strings
func (n *NICInfo) MarshalJSON() ([]byte, error) {
	var mac, ip string
	mac = n.MAC.String()
	if n.IP != nil {
		ip = n.IP.String()
	}
	info := &stringNICInfo{MAC: mac, IP: ip}
	return json.Marshal(&info)
}

// UnmarshalJSON unmarshals a NICInfo object, converting IP and MAC from strings
func (n *NICInfo) UnmarshalJSON(data []byte) error {
	rawData := &stringNICInfo{}
	err := json.Unmarshal(data, rawData)
	if err != nil {
		return err
	}

	err = rawData.populateNICInfo(n)
	return err
}

// UnmarshalYAML unmarshals a NICInfo object, converting IP and MAC from strings
func (n *NICInfo) UnmarshalYAML(unmarshal func(interface{}) error) error {
	rawData := &stringNICInfo{}
	err := unmarshal(rawData)
	if err != nil {
		return err
	}

	err = rawData.populateNICInfo(n)
	return err
}

type NetworkInterface struct {
	NICs     []net.HardwareAddr `json:"-" dynamodbav:"nics"`
	Metadata Metadata
}

// MarshalJSON marshals a NICInfo object, converting IP and MAC to strings
func (n *NetworkInterface) MarshalJSON() ([]byte, error) {
	type Alias NetworkInterface
	v := &struct {
		*Alias
		NICs []string `json:"nics"`
	}{
		Alias: (*Alias)(n),
	}

	v.NICs = make([]string, 0, len(n.NICs))
	for _, mac := range n.NICs {
		v.NICs = append(v.NICs, mac.String())
	}

	if v.Metadata == nil {
		v.Metadata = make(Metadata)
	}
	return json.Marshal(v)
}

// UnmarshalJSON unmarshals a NICInfo object, converting IP and MAC from strings
func (n *NetworkInterface) UnmarshalJSON(data []byte) error {
	type Alias NetworkInterface
	v := &struct {
		*Alias
		NICs []string `json:"nics"`
	}{
		Alias: (*Alias)(n),
	}
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	if n.Metadata == nil {
		n.Metadata = make(Metadata)
	}

	n.NICs = make([]net.HardwareAddr, 0, len(v.NICs))
	for _, macString := range v.NICs {
		mac, err := net.ParseMAC(macString)
		if err != nil {
			return fmt.Errorf("unable to parse mac '%s': %v", macString, err)
		}
		n.NICs = append(n.NICs, mac)
	}
	return nil
}

func (n *NetworkInterface) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	type Alias NetworkInterface
	iface := &Alias{}

	// check for NicInfo
	if macAv, ok := av.M["MAC"]; ok {
		av.M["NICs"] = &dynamodb.AttributeValue{L: []*dynamodb.AttributeValue{macAv}}
		delete(av.M, "MAC")
	}

	err := dynamodbattribute.Unmarshal(av, iface)
	if err != nil {
		return err
	}
	*n = (NetworkInterface)(*iface)
	return nil
}
