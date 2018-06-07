package types

import (
	"encoding/json"
	"testing"
)

func getTestNICInstance() (*NICInstance, string) {
	nicInfo, _, _ := getTestNICInfo()
	net, _, _ := getTestNetwork()
	nicInstance := &NICInstance{NIC: *nicInfo, Network: *net, Config: NicConfig{IP: []string{"10.0.0.1/24"}, Gateway: []string{"10.0.0.254"}, DNS: []string{"10.53.53.53"}}}
	jsonString := `{"Network":{"Name":"test_phys","MTU":9000,"Subnets":[{"Name":"testsubnet","Gateway":"10.0.0.254","DNS":["10.53.53.53"],"AllocationMethod":"","Cidr":"10.0.0.0/24"}],"Domain":"test.local","Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"},"NIC":{"MAC":"00:02:03:04:05:06","IP":"10.0.0.1"},"Config":{"IP":["10.0.0.1/24"],"Gateway":["10.0.0.254"],"DNS":["10.53.53.53"]}}`
	return nicInstance, jsonString
}

func TestNICInstanceMarshalJSON(t *testing.T) {
	nic, jsonstring := getTestNICInstance()
	actualString, err := json.Marshal(nic)
	if err != nil {
		t.Fatalf("Unable to marshal: %v", err)
	}

	if string(actualString) != jsonstring {
		t.Fatalf("Got: %s, Expected: %s", string(actualString), jsonstring)
	}
}

func TestNICInstanceUnmarshalJSON(t *testing.T) {
	expected, jsonString := getTestNICInstance()
	nic := &NICInstance{}
	testUnmarshalJSON(t, nic, expected, jsonString)
}
