package types

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/azenk/iputils"
)

type IPReservation struct {
	HostInformation string
	IP              *net.IPNet       `json:"ip"`
	MAC             net.HardwareAddr `json:"mac"`
	Gateway         net.IP           `json:"gateway"`
	DNS             []net.IP         `json:"dns"`
	Start           *time.Time       `json:"start"`
	End             *time.Time       `json:"end"`
	Metadata        Metadata         `json:"metadata"`
}

func NewStaticIPReservation() *IPReservation {
	now := time.Now()
	return &IPReservation{Start: &now, Metadata: make(Metadata)}
}

func NewDynamicIPReservation(ttl time.Duration) *IPReservation {
	now := time.Now()
	end := now.Add(ttl)
	return &IPReservation{Start: &now, End: &end, Metadata: make(Metadata)}
}

func (r *IPReservation) SetRandomIP() error {
	startOffset, ipLength := r.IP.Mask.Size()
	networkIP, _ := iputils.SetBits(r.IP.IP, uint64(0), uint(startOffset), uint(ipLength-startOffset))
	broadcastIP, _ := iputils.SetBits(r.IP.IP, uint64(0xffffffffffffffff), uint(startOffset), uint(ipLength-startOffset))

	rand.Seed(time.Now().UnixNano())
	var allocatedIP *net.IPNet
	maxCount := 1 << uint(ipLength-startOffset)
	for count := 0; allocatedIP == nil && count < maxCount; count++ {
		// choose IP at random until we find a free one
		randomHostPart := rand.Uint64()
		candidateIP, err := iputils.SetBits(r.IP.IP, randomHostPart, uint(startOffset), uint(ipLength-startOffset))
		if err != nil {
			return fmt.Errorf("unexpected error building ip: %v", err)
		}
		if candidateIP.To4() != nil && (candidateIP.Equal(networkIP) || candidateIP.Equal(broadcastIP)) {
			continue
		}
		allocatedIP = &net.IPNet{IP: candidateIP, Mask: r.IP.Mask}
	}
	r.IP = allocatedIP
	return nil
}

// Validate checks that the reservation is internally consistent and fully populated
func (r *IPReservation) Validate() bool {
	if r.IP == nil {
		return false
	}

	if r.IP.IP.To4() != nil {
		startOffset, ipLength := r.IP.Mask.Size()
		networkIP := r.IP.IP.Mask(r.IP.Mask)
		broadcastIP, _ := iputils.SetBits(networkIP, uint64(0xffffffffffffffff), uint(startOffset), uint(ipLength-startOffset))
		return !r.IP.IP.Equal(networkIP) && !r.IP.IP.Equal(broadcastIP)
	} else {
		return true
	}
}

// ValidAt returns true if an IPReservation is valid at the time specified
func (r *IPReservation) ValidAt(t time.Time) bool {
	if r.Start != nil && r.Start.After(t) {
		return false
	}

	if r.End != nil && r.End.Before(t) {
		return false
	}

	return true
}

func (r *IPReservation) Static() bool {
	return r.End == nil
}

func (r *IPReservation) SetSubnetInformation(subnet *Subnet) {
	r.Gateway = nil
	if subnet.Gateway != nil {
		r.Gateway = subnet.Gateway
	}

	r.DNS = []net.IP{}
	for _, dns := range subnet.DNS {
		r.DNS = append(r.DNS, dns)
	}
}

func (r *IPReservation) MarshalJSON() ([]byte, error) {
	type Alias IPReservation
	v := &struct {
		*Alias
		IP      string   `json:"ip"`
		MAC     string   `json:"mac"`
		Gateway string   `json:"gateway"`
		DNS     []string `json:"dns"`
	}{
		Alias: (*Alias)(r),
	}
	v.MAC = v.Alias.MAC.String()
	v.IP = v.Alias.IP.String()
	v.DNS = []string{}
	for _, dns := range v.Alias.DNS {
		v.DNS = append(v.DNS, dns.String())
	}
	if v.Alias.Gateway != nil {
		v.Gateway = v.Alias.Gateway.String()
	}
	return json.Marshal(v)
}

// UnmarshalJSON implements Unmarshaler interface so that cidr can be directly
// read from a string
func (r *IPReservation) UnmarshalJSON(data []byte) error {
	type Alias IPReservation
	v := &struct {
		*Alias
		IP      string   `json:"ip"`
		MAC     string   `json:"mac"`
		Gateway string   `json:"gateway"`
		DNS     []string `json:"dns"`
	}{
		Alias: (*Alias)(r),
	}
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	if v.IP != "" {
		ip, cidr, err := net.ParseCIDR(v.IP)
		if err != nil {
			return err
		}
		cidr.IP = ip
		r.IP = cidr
		if err != nil {
			return err
		}
	}

	if v.MAC != "" {
		mac, err := net.ParseMAC(v.MAC)
		if err != nil {
			return err
		}
		r.MAC = mac
	}

	r.Gateway = net.ParseIP(v.Gateway)

	for _, dns := range v.DNS {
		ip := net.ParseIP(dns)
		if ip != nil {
			r.DNS = append(r.DNS, ip)
		}
	}
	if r.Metadata == nil {
		r.Metadata = make(Metadata)
	}
	return err
}
func (r *IPReservation) MarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	av.M = make(map[string]*dynamodb.AttributeValue, 0)

	v, err := dynamodbattribute.Marshal(r.HostInformation)
	if err != nil {
		return err
	}
	av.M["HostInformation"] = v

	if r.MAC != nil {
		m, err := dynamodbattribute.Marshal(r.MAC.String())
		if err != nil {
			return err
		}
		av.M["MAC"] = m
	}

	if r.IP != nil {
		i, err := dynamodbattribute.Marshal(r.IP.String())
		if err != nil {
			return err
		}
		av.M["IP"] = i
	}

	if r.Start != nil {
		s, err := dynamodbattribute.Marshal(r.Start.Unix())
		if err != nil {
			return err
		}
		av.M["Start"] = s
	}

	if r.End != nil {
		e, err := dynamodbattribute.Marshal(r.End.Unix())
		if err != nil {
			return err
		}
		av.M["End"] = e
	}

	metadataAv, err := dynamodbattribute.Marshal(r.Metadata)
	if err != nil {
		return err
	}
	av.M["Metadata"] = metadataAv
	return nil
}

func (r *IPReservation) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	if v, ok := av.M["HostInformation"]; ok && v.NULL == nil {
		r.HostInformation = v.String()
	}

	if v, ok := av.M["MAC"]; ok && v.NULL == nil {
		mac, err := net.ParseMAC(*v.S)
		if err != nil {
			return err
		}
		r.MAC = mac
	}

	if v, ok := av.M["IP"]; ok && v.NULL == nil {
		i, n, err := net.ParseCIDR(*v.S)
		if err != nil {
			return err
		}
		n.IP = i
		r.IP = n
	}

	if v, ok := av.M["Start"]; ok && v.NULL == nil {
		sEpoch := new(int64)
		err := dynamodbattribute.Unmarshal(v, sEpoch)
		if err != nil {
			return fmt.Errorf("unable to unmarshal start time: %v", err)
		}
		s := time.Unix(*sEpoch, 0)
		r.Start = &s
	}

	if v, ok := av.M["End"]; ok && v.NULL == nil {
		eEpoch := new(int64)
		err := dynamodbattribute.Unmarshal(v, eEpoch)
		if err != nil {
			return fmt.Errorf("unable to unmarshal end time: %v", err)
		}
		e := time.Unix(*eEpoch, 0)
		r.End = &e
	}

	if v, ok := av.M["Metadata"]; ok && v.NULL == nil {
		metadata := make(Metadata)
		err := dynamodbattribute.Unmarshal(v, metadata)
		if err != nil {
			return fmt.Errorf("unable to unmarshal end time: %v", err)
		}
		r.Metadata = metadata
	} else {
		r.Metadata = make(Metadata)
	}

	return nil
}

type IPReservationList []*IPReservation

func (l IPReservationList) Static() IPReservationList {
	staticList := IPReservationList{}
	for _, r := range l {
		if r.Static() {
			staticList = append(staticList, r)
		}
	}
	return staticList
}

func (l IPReservationList) Dynamic() IPReservationList {
	dynamicList := IPReservationList{}
	for _, r := range l {
		if !r.Static() {
			dynamicList = append(dynamicList, r)
		}
	}
	return dynamicList
}

func (l IPReservationList) ValidAt(t time.Time) IPReservationList {
	result := IPReservationList{}
	for _, r := range l {
		if r.ValidAt(t) {
			result = append(result, r)
		}
	}
	return result
}

func (l IPReservationList) Contains(ip net.IP) bool {
	for _, r := range l {
		if r.IP.IP.Equal(ip) {
			return true
		}
	}
	return false
}
