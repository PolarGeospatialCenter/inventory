package types

import (
	"encoding/json"
	"net"
	"testing"
)

func getTestSubnetV4() (*Subnet, string, string) {
	gateway := net.ParseIP("10.0.0.254")
	_, cidr, _ := net.ParseCIDR("10.0.0.0/24")
	dns0 := net.ParseIP("10.0.1.1")
	dns1 := net.ParseIP("10.0.2.2")
	dns := []net.IP{dns0, dns1}
	subnet := &Subnet{Name: "test", Cidr: cidr, Gateway: gateway, DNS: dns, AllocationMethod: "static_inventory"}
	marshaledJSON := `{"Name":"test","Gateway":"10.0.0.254","DNS":["10.0.1.1","10.0.2.2"],"AllocationMethod":"static_inventory","Cidr":"10.0.0.0/24"}`
	marhsaledYAML := `name: "test"
cidr: 10.0.0.0/24
gateway: 10.0.0.254
allocationmethod: static_inventory
dns:
  - 10.0.1.1
  - 10.0.2.2
`
	return subnet, marshaledJSON, marhsaledYAML
}

func TestSubnetMarshalV4(t *testing.T) {
	i, expected, _ := getTestSubnetV4()
	result, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nic info %v", i)
	}

	if string(result) != expected {
		t.Fatalf("The marshaled version of the Subnet is incorrect: '%s', Expected: '%s'", string(result), expected)
	}
}

func TestSubnetUnmarshalJSONV4(t *testing.T) {
	expected, testText, _ := getTestSubnetV4()
	subnet := &Subnet{}
	testUnmarshalJSON(t, subnet, expected, testText)
}

func TestSubnetUnmarshalYAMLV4(t *testing.T) {
	expected, _, testText := getTestSubnetV4()
	subnet := &Subnet{}
	testUnmarshalYAML(t, subnet, expected, testText)
}

func getTestSubnetV6() (*Subnet, string, string) {
	gateway := net.ParseIP("2001:db8:0:1::1")
	_, cidr, _ := net.ParseCIDR("2001:db8:0:1::/64")
	dns0 := net.ParseIP("2001:db8:0:2::1")
	dns1 := net.ParseIP("10.0.2.2")
	dns := []net.IP{dns0, dns1}
	subnet := &Subnet{Name: "test", Cidr: cidr, Gateway: gateway, DNS: dns}
	marshaledJSON := `{"Name":"test","Gateway":"2001:db8:0:1::1","DNS":["2001:db8:0:2::1","10.0.2.2"],"AllocationMethod":"","Cidr":"2001:db8:0:1::/64"}`
	marhsaledYAML := `name: "test"
cidr: 2001:db8:0:1::/64
gateway: 2001:db8:0:1::1
dns:
  - 2001:db8:0:2::1
  - 10.0.2.2
`
	return subnet, marshaledJSON, marhsaledYAML
}
func TestSubnetMarshalV6(t *testing.T) {
	i, expected, _ := getTestSubnetV6()
	result, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nic info %v", i)
	}

	if string(result) != expected {
		t.Fatalf("The marshaled version of the Subnet is incorrect: '%s', Expected: '%s'", string(result), expected)
	}
}

func TestSubnetUnmarshalJSONV6(t *testing.T) {
	expected, testText, _ := getTestSubnetV6()
	subnet := &Subnet{}
	testUnmarshalJSON(t, subnet, expected, testText)
}

func TestSubnetUnmarshalYAMLV6(t *testing.T) {
	expected, _, testText := getTestSubnetV6()
	subnet := &Subnet{}
	testUnmarshalYAML(t, subnet, expected, testText)
}
