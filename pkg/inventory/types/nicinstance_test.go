package types

import (
	"encoding/json"
	"testing"
)

func getTestNICInstance() (*NICInstance, string) {
	nicInfo, _, _ := getTestNICInfo()
	net, _, _ := getTestNetwork()
	nicInstance := &NICInstance{NIC: *nicInfo, Network: *net}
	jsonString := `{"Network":{"Name":"test_phys","MTU":9000,"Subnets":[],"Domain":"test.local","Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T15:33:09-06:00"},"NIC":{"MAC":"00:02:03:04:05:06","IP":"10.0.0.1"}}`
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
