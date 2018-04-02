package types

import (
	"encoding/json"
	"net"
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
