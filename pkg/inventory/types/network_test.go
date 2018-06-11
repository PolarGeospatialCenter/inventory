package types

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func getTestNetwork() (*Network, string, string) {
	network := NewNetwork()
	network.Domain = "test.local"
	network.MTU = 9000
	network.Name = "test_phys"
	_, cidr, _ := net.ParseCIDR("10.0.0.0/24")
	network.Subnets = []*Subnet{
		&Subnet{Name: "testsubnet", Cidr: cidr, Gateway: net.ParseIP("10.0.0.254"), DNS: []net.IP{net.ParseIP("10.53.53.53")}},
	}

	network.LastUpdated = time.Unix(123456789, 0).UTC()
	network.Metadata = make(map[string]interface{})
	network.Metadata["foo"] = "test"
	network.Metadata["bar"] = 34.1

	jsonString := `{"Name":"test_phys","MTU":9000,"Subnets":[{"Name":"testsubnet","Gateway":"10.0.0.254","DNS":["10.53.53.53"],"AllocationMethod":"","Cidr":"10.0.0.0/24"}],"Domain":"test.local","Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"}`
	yamlString := `name: test_phys
mtu: 9000
domain: test.local
subnets:
  - name: testsubnet
    cidr: "10.0.0.0/24"
    dns:
      - 10.53.53.53
    gateway: 10.0.0.254
lastupdated: 1973-11-29T21:33:09Z
metadata:
  foo: test
  bar: 34.1
`
	return network, jsonString, yamlString
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
