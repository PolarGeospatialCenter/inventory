package types

import "net"

type NicConfig struct {
	IP      []string
	Gateway []string
	DNS     []string
}

func NewNicConfig() *NicConfig {
	return &NicConfig{
		IP:      make([]string, 0),
		DNS:     make([]string, 0),
		Gateway: make([]string, 0),
	}
}

func (c *NicConfig) Append(ip net.IPNet, dns []net.IP, gateway *net.IP) {
	c.IP = append(c.IP, ip.String())
	for _, dnsIP := range dns {
		c.DNS = append(c.DNS, dnsIP.String())
	}
	if gateway != nil {
		c.Gateway = append(c.Gateway, gateway.String())
	}
}
