package types

import (
	"fmt"
	"net"
	"time"
)

type IpamIpRequest struct {
	Subnet    string   `json:"subnet"`
	HwAddress string   `json:"mac"`
	TTL       string   `json:"ttl"`
	Metadata  Metadata `json:"metadata"`
}

func (req *IpamIpRequest) Reservation(ip net.IP) (*IPReservation, error) {
	r := &IPReservation{}

	if req.Subnet == "" && ip == nil {
		return nil, fmt.Errorf("must specify a subnet or IP address to create a reservation")
	}

	if req.HwAddress != "" {
		mac, err := net.ParseMAC(req.HwAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid MAC address")
		}
		r.MAC = mac
	}

	start := time.Now()
	r.Start = &start

	if req.TTL != "" {
		ttl, err := time.ParseDuration(req.TTL)
		if err != nil {
			return nil, fmt.Errorf("if a ttl is provided it must be a golang duration string")
		}
		end := start.Add(ttl)
		r.End = &end
	}

	if req.Subnet != "" && ip == nil && r.MAC == nil && r.End == nil {
		return nil, fmt.Errorf("all dynamic reservations must include a MAC address or a TTL")
	}

	r.Metadata = make(Metadata, len(req.Metadata))
	for k, v := range req.Metadata {
		r.Metadata[k] = v
	}

	return r, nil
}
