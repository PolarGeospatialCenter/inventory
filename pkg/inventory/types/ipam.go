package types

import (
	"net"
)


type IpamIpRequest struct {
	Name      string `json:"name"`
	Subnet    string `json:"subnet"`
	HwAddress string `json:"mac"`
	TTL       string `json:"ttl"`
}

type IpamIpResponse struct {
	IPReservation
	Gateway net.IP
	DNS     []net.IP
}


