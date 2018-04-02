package types

import (
	"encoding/json"
	"testing"
	"time"
)

func getTestNetwork() (*Network, string, string) {
	net := NewNetwork()
	net.Domain = "test.local"
	net.MTU = 9000
	net.Name = "test_phys"
	net.Subnets = make([]*Subnet, 0)
	net.LastUpdated = time.Unix(123456789, 0)
	net.Metadata = make(map[string]interface{})
	net.Metadata["foo"] = "test"
	net.Metadata["bar"] = 34.1

	jsonString := `{"Name":"test_phys","MTU":9000,"Subnets":[],"Domain":"test.local","Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T15:33:09-06:00"}`
	yamlString := `name: test_phys
mtu: 9000
domain: test.local
subnets: []
lastupdated: 1973-11-29T15:33:09-06:00
metadata:
  foo: test
  bar: 34.1
`
	return net, jsonString, yamlString
}

func TestNetworkMarshalJSON(t *testing.T) {
	net, jsonstring, _ := getTestNetwork()
	actualString, err := json.Marshal(net)
	if err != nil {
		t.Fatalf("Unable to marshal: %v", err)
	}

	if string(actualString) != jsonstring {
		t.Fatalf("Got: %s, Expected: %s", string(actualString), jsonstring)
	}
}

func TestNetworkUnmarshalJSON(t *testing.T) {
	expected, jsonString, _ := getTestNetwork()
	net := &Network{}
	testUnmarshalJSON(t, net, expected, jsonString)
}

func TestNetworkUnmarshalYAML(t *testing.T) {
	expected, _, yamlString := getTestNetwork()
	net := &Network{}
	testUnmarshalYAML(t, net, expected, yamlString)
}
