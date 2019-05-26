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
	IP              *net.IPNet
	MAC             net.HardwareAddr
	Start           *time.Time
	End             *time.Time
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

func (r *IPReservation) MarshalJSON() ([]byte, error) {
	type Alias IPReservation
	v := &struct {
		*Alias
		IP  string
		MAC string
	}{
		Alias: (*Alias)(r),
	}
	v.MAC = v.Alias.MAC.String()
	v.IP = v.Alias.IP.String()
	return json.Marshal(v)
}

// UnmarshalJSON implements Unmarshaler interface so that cidr can be directly
// read from a string
func (r *IPReservation) UnmarshalJSON(data []byte) error {
	type Alias IPReservation
	v := &struct {
		*Alias
		IP  string
		MAC string
	}{
		Alias: (*Alias)(r),
	}
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	ip, cidr, err := net.ParseCIDR(v.IP)
	if err != nil {
		return err
	}
	cidr.IP = ip
	r.IP = cidr
	if err != nil {
		return err
	}

	mac, err := net.ParseMAC(v.MAC)
	r.MAC = mac
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
		var sEpoch *int64
		err := dynamodbattribute.Unmarshal(v, sEpoch)
		if err != nil {
			return err
		}
		s := time.Unix(*sEpoch, 0)
		r.Start = &s
	}

	if v, ok := av.M["End"]; ok && v.NULL == nil {
		var eEpoch *int64
		err := dynamodbattribute.Unmarshal(v, eEpoch)
		if err != nil {
			return err
		}
		e := time.Unix(*eEpoch, 0)
		r.Start = &e
	}

	return nil
}
