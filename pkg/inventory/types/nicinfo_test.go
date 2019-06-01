package types

import (
	"encoding/json"
	"net"
	"testing"
)

func getTestNICInfo() (*NetworkInterface, string) {
	mac, _ := net.ParseMAC("00:02:03:04:05:06")
	nicInfo := &NetworkInterface{NICs: []net.HardwareAddr{mac}, Metadata: Metadata{}}
	marshaledJSON := `{"Metadata":{},"nics":["00:02:03:04:05:06"]}`
	return nicInfo, marshaledJSON
}

func TestNICInfoMarshal(t *testing.T) {
	i, expected := getTestNICInfo()
	result, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nic info %v", i)
	}

	if string(result) != expected {
		t.Fatalf("The marshaled version of the NICInfo is incorrect: '%s', Expected: '%s'", string(result), expected)
	}

	i = &NetworkInterface{}
	result, err = json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nil nic info: %v", err)
	}

	if string(result) != `{"Metadata":{},"nics":[]}` {
		t.Fatalf("Marshaled version of nil NICInfo incorrect: %s", string(result))
	}
}

func TestNICInfoUnmarshalJSON(t *testing.T) {
	expected, testText := getTestNICInfo()
	info := &NetworkInterface{}
	testUnmarshalJSON(t, info, expected, testText)
}
