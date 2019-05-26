package types

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type IPReservation struct {
	HostInformation string
	IP              *net.IPNet
	MAC             net.HardwareAddr
	Start           *time.Time
	End             *time.Time
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

	s, err := dynamodbattribute.Marshal(r.Start)
	if err != nil {
		return err
	}
	av.M["Start"] = s

	e, err := dynamodbattribute.Marshal(r.End)
	if err != nil {
		return err
	}
	av.M["End"] = e
	return nil
}

func (r *IPReservation) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	if v, ok := av.M["HostInformation"]; ok {
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

	if v, ok := av.M["Start"]; ok {
		log.Print(v)
	}

	if v, ok := av.M["End"]; ok {
		log.Print(v)
	}

	return nil
}
