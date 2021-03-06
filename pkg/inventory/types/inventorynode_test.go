package types

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/go-test/deep"
)

func getTestInventoryNode() (*InventoryNode, string, error) {
	node, _ := getTestNode()
	sys, _, _ := getTestSystem()
	network, _ := getTestNetwork()
	mac, _ := net.ParseMAC("00:02:03:04:05:06")
	ipreservationdb := IPReservationMap{
		"00:02:03:04:05:06": IPReservationList{
			&IPReservation{
				IP:  &net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0)},
				MAC: mac,
			},
		},
	}
	v6subnet := &Subnet{Name: "testv6", Cidr: &net.IPNet{IP: net.ParseIP("2001:db8::"), Mask: net.CIDRMask(64, 128)}, StaticAllocationMethod: "random", Gateway: net.ParseIP("2001:db8::1")}
	network.Subnets = append(network.Subnets, v6subnet)
	networks := make(NetworkMap)
	networks[network.ID()] = network
	systems := make(SystemMap)
	systems[sys.ID()] = sys
	jsonString := `{"Hostname":"test-te12-04-a","LocationString":"te12-04-a","InventoryID":"sample0002","Tags":["foo","bar","baz"],"Networks":{"logical":{"Network":{"Name":"test_phys","MTU":9000,"Subnets":[{"Name":"testsubnet","Gateway":"10.0.0.254","DNS":["10.53.53.53"],"StaticAllocationMethod":"","DynamicAllocationMethod":"","Cidr":"10.0.0.0/24"},{"Name":"testv6","Gateway":"2001:db8::1","DNS":null,"StaticAllocationMethod":"random","DynamicAllocationMethod":"","Cidr":"2001:db8::/64"}],"Domain":"test.local","Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"},"Interface":{"Metadata":{},"nics":["00:02:03:04:05:06"]},"Config":{"IP":["10.0.0.1/24"],"Gateway":["10.0.0.254"],"DNS":["10.53.53.53"]}}},"Role":"worker","Location":{"Building":"123 Fake St","Room":"305","Rack":"te12","BottomU":4},"ChassisSubIndex":"a","System":{"Name":"Test System","ShortName":"test","Environments":{"production":{"IPXEUrl":"http://localhost:8080/","Networks":{"logical":"test_phys"},"Metadata":{"key":"value"}}},"Roles":["worker"],"Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"},"Environment":{"IPXEUrl":"http://localhost:8080/","Networks":{"logical":"test_phys"},"Metadata":{"key":"value"}},"Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"}`
	inode, err := NewInventoryNode(node, networks, systems, ipreservationdb)
	return inode, jsonString, err
}

func TestInventoryNodeMarshalJSON(t *testing.T) {
	node, jsonstring, err := getTestInventoryNode()
	if err != nil {
		t.Fatalf("Unable to build inventory node: %v", err)
	}
	actualString, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Unable to marshal: %v", err)
	}

	if string(actualString) != jsonstring {
		t.Fatalf("Got: %s, Expected: %s", string(actualString), jsonstring)
	}
}

func TestInventoryNodeUnmarshalJSON(t *testing.T) {
	expected, jsonString, _ := getTestInventoryNode()
	node := &InventoryNode{}
	testUnmarshalJSON(t, node, expected, jsonString)
}

func TestInventoryNodeIPs(t *testing.T) {
	node, _, err := getTestInventoryNode()
	if err != nil {
		t.Fatalf("Unable to create inventory node for testing: %v", err)
	}
	expected_ips := []net.IP{net.ParseIP("10.0.0.1")}

	if diff := deep.Equal(node.IPs(), expected_ips); diff != nil {
		t.Errorf("IP list not equal to expected:")
		for _, d := range diff {
			t.Error(d)
		}
		t.FailNow()
	}

}
