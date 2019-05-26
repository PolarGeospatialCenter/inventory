package types

import (
	"fmt"
	"log"
	"net"
	"time"
)

type IpamIpRequest struct {
	Name      string `json:"name"`
	Subnet    string `json:"subnet"`
	HwAddress string `json:"mac"`
	TTL       string `json:"ttl"`
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
			log.Printf("unable to parse provided ttl '%s': %v", req.TTL, err)
			return nil, fmt.Errorf("if a ttl is provided it must be a golang duration string")
		}
		end := start.Add(ttl)
		r.End = &end
	}

	if req.Subnet != "" && ip == nil && r.MAC == nil && r.End == nil {
		return nil, fmt.Errorf("all dynamic reservations must include a MAC address or a TTL")
	}

	return r, nil
}

type IpamIpResponse struct {
	IP              string
	MAC             string
	HostInformation string
	Gateway         string
	DNS             []string
	Start           *time.Time
	End             *time.Time
}

func NewIpamResponseFromReservation(reservation *IPReservation) *IpamIpResponse {
	response := &IpamIpResponse{}

	if reservation.IP != nil {
		response.IP = reservation.IP.String()
	}

	if reservation.MAC != nil {
		response.MAC = reservation.MAC.String()
	}

	response.HostInformation = reservation.HostInformation
	response.Start = reservation.Start
	response.End = reservation.End
	return response
}

func (r *IpamIpResponse) SetSubnetInformation(subnet *Subnet) {
	r.Gateway = ""
	if subnet.Gateway != nil {
		r.Gateway = subnet.Gateway.String()
	}

	r.DNS = []string{}
	for _, dns := range subnet.DNS {
		r.DNS = append(r.DNS, dns.String())
	}
}
