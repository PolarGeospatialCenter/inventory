package types

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/go-test/deep"
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

func TestNetworkSetTimestamp(t *testing.T) {
	n := &Network{}
	ts := time.Now()
	n.SetTimestamp(ts)
	if n.Timestamp() != ts.Unix() {
		t.Errorf("Timestamp returned doesn't match the time set.")
	}
}

func TestDynamoDBRoundTrip(t *testing.T) {
	expected, _, _ := getTestNetwork()

	dynamodbValue, err := dynamodbattribute.Marshal(expected)
	if err != nil {
		t.Errorf("unable to marshal network to dynamodb attribute: %v", err)
	}

	unmarshaledNetwork := &Network{}
	err = dynamodbattribute.Unmarshal(dynamodbValue, unmarshaledNetwork)
	if err != nil {
		t.Errorf("unable to unmarshal network from dynamodb attribute: %v", err)
	}

	if diff := deep.Equal(unmarshaledNetwork, expected); len(diff) > 0 {
		t.Error("Unmarshaled object not equal to expected:")
		for _, l := range diff {
			t.Error(l)
		}
	}
}

func TestDynamoDBRoundTripNoSubnets(t *testing.T) {
	expected, _, _ := getTestNetwork()

	expected.Subnets = []*Subnet{}

	dynamodbValue, err := dynamodbattribute.Marshal(expected)
	if err != nil {
		t.Errorf("unable to marshal network to dynamodb attribute: %v", err)
	}

	unmarshaledNetwork := &Network{}
	err = dynamodbattribute.Unmarshal(dynamodbValue, unmarshaledNetwork)
	if err != nil {
		t.Errorf("unable to unmarshal network from dynamodb attribute: %v", err)
	}

	if diff := deep.Equal(unmarshaledNetwork, expected); len(diff) > 0 {
		t.Error("Unmarshaled object not equal to expected:")
		for _, l := range diff {
			t.Error(l)
		}
	}
}
