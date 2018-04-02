package types

import (
	"encoding/json"
	"net"
	"testing"
)

func getTestNICInfo() (*NICInfo, string, string) {
	mac, _ := net.ParseMAC("00:02:03:04:05:06")
	ip := net.ParseIP("10.0.0.1")
	nicInfo := &NICInfo{MAC: mac, IP: ip}
	marshaledJSON := "{\"MAC\":\"00:02:03:04:05:06\",\"IP\":\"10.0.0.1\"}"
	marhsaledYAML := "mac: 00:02:03:04:05:06\nip: 10.0.0.1\n"
	return nicInfo, marshaledJSON, marhsaledYAML
}

func TestNicInfoUnmarshalNoMac(t *testing.T) {
	ip := net.ParseIP("10.0.0.1")
	nicInfo := &NICInfo{MAC: net.HardwareAddr{}, IP: ip}
	marshaledJSON := "{\"IP\":\"10.0.0.1\"}"
	marhsaledYAML := "ip: 10.0.0.1\n"

	testUnmarshalYAML(t, &NICInfo{}, nicInfo, marhsaledYAML)
	testUnmarshalJSON(t, &NICInfo{}, nicInfo, marshaledJSON)
}

func TestNICInfoMarshal(t *testing.T) {
	i, expected, _ := getTestNICInfo()
	result, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nic info %v", i)
	}

	if string(result) != expected {
		t.Fatalf("The marshaled version of the NICInfo is incorrect: '%s', Expected: '%s'", string(result), expected)
	}

	i = &NICInfo{}
	result, err = json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nil nic info: %v", err)
	}

	if string(result) != "{\"MAC\":\"\",\"IP\":\"\"}" {
		t.Fatalf("Marshaled version of nil NICInfo incorrect: %s", string(result))
	}
}

func TestNICInfoUnmarshalJSON(t *testing.T) {
	expected, testText, _ := getTestNICInfo()
	info := &NICInfo{}
	testUnmarshalJSON(t, info, expected, testText)
}

func TestNICInfoUnmarshalYAML(t *testing.T) {
	expected, _, testText := getTestNICInfo()
	info := &NICInfo{}
	testUnmarshalYAML(t, info, expected, testText)
}
