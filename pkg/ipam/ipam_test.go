package ipam

import (
	"net"
	"testing"

	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/go-test/deep"
)

func TestIsV6(t *testing.T) {
	v4Ip := net.ParseIP("10.0.0.0")
	v6Ip := net.ParseIP("2001:db8::1")

	if IsV6(v4Ip) {
		t.Errorf("improperly identified v4 address as ipv6")
	}

	if !IsV6(v6Ip) {
		t.Errorf("improperly identified v6 address as not ipv6")
	}
}

func TestIPv6IPAllocation(t *testing.T) {
	type testCase struct {
		Subnet          *inventorytypes.Subnet
		Location        *inventorytypes.ChassisLocation
		ChassisSubIndex string
		ExpectedIp      net.IP
		ExpectedErr     error
	}

	_, v6cidrA, _ := net.ParseCIDR("2001:db8::/64")
	_, v6cidrB, _ := net.ParseCIDR("2001:db8::/56")
	_, v4cidr, _ := net.ParseCIDR("10.0.0.0/24")

	cases := []*testCase{
		&testCase{
			Subnet:          &inventorytypes.Subnet{Cidr: v6cidrB},
			Location:        &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31},
			ChassisSubIndex: "a",
			ExpectedIp:      net.ParseIP("2001:db8::e0:1ce1:fa00:0:1"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          &inventorytypes.Subnet{Cidr: v6cidrA},
			Location:        &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31},
			ChassisSubIndex: "a",
			ExpectedIp:      net.ParseIP("2001:db8::e01c:e1fa:0:1"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          &inventorytypes.Subnet{Cidr: v4cidr},
			Location:        &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31},
			ChassisSubIndex: "a",
			ExpectedIp:      net.IP{},
			ExpectedErr:     ErrAllocationNotImplemented,
		},
	}

	for _, c := range cases {
		ip, err := GetIPByLocation(c.Subnet, c.Location, c.ChassisSubIndex)
		if err != c.ExpectedErr {
			t.Errorf("got unexpected error: Expected: %v -- Got: %v", c.ExpectedErr, err)
		}

		if diff := deep.Equal(ip, c.ExpectedIp); len(diff) > 0 {
			t.Errorf("got incorrect IP:")
			for _, l := range diff {
				t.Error(l)
			}
		}
	}
}

func TestIPv6RangeAllocation(t *testing.T) {
	type testCase struct {
		Subnet          *inventorytypes.Subnet
		Location        *inventorytypes.ChassisLocation
		ChassisSubIndex string
		ExpectedStartIp net.IP
		ExpectedEndIp   net.IP
		ExpectedErr     error
	}

	_, v6cidrA, _ := net.ParseCIDR("2001:db8::/64")
	_, v6cidrB, _ := net.ParseCIDR("2001:db8::/56")
	_, v4cidr, _ := net.ParseCIDR("10.0.0.0/24")

	cases := []*testCase{
		&testCase{
			Subnet:          &inventorytypes.Subnet{Cidr: v6cidrB},
			Location:        &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31},
			ChassisSubIndex: "a",
			ExpectedStartIp: net.ParseIP("2001:db8:0:60:1ce1:fa00::"),
			ExpectedEndIp:   net.ParseIP("2001:db8:0:60:1ce1:faff:ffff:ffff"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          &inventorytypes.Subnet{Cidr: v6cidrA},
			Location:        &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31},
			ChassisSubIndex: "a",
			ExpectedStartIp: net.ParseIP("2001:db8:0:0:601c:e1fa:0::"),
			ExpectedEndIp:   net.ParseIP("2001:db8::601c:e1fa:ffff:ffff"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          &inventorytypes.Subnet{Cidr: v4cidr},
			Location:        &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31},
			ChassisSubIndex: "a",
			ExpectedStartIp: net.IP{},
			ExpectedEndIp:   net.IP{},
			ExpectedErr:     ErrAllocationNotImplemented,
		},
	}

	for _, c := range cases {
		t.Logf("Testing %s: %s", c.Subnet.Cidr.String(), c.ExpectedStartIp)
		ipStart, ipEnd, err := GetRangeByLocation(c.Subnet, c.Location, c.ChassisSubIndex)
		if err != c.ExpectedErr {
			t.Errorf("got unexpected error: Expected: %v -- Got: %v", c.ExpectedErr, err)
		}

		if diff := deep.Equal(ipStart, c.ExpectedStartIp); len(diff) > 0 {
			t.Errorf("got incorrect IP:")
			for _, l := range diff {
				t.Error(l)
			}
		}

		if diff := deep.Equal(ipEnd, c.ExpectedEndIp); len(diff) > 0 {
			t.Errorf("got incorrect IP:")
			for _, l := range diff {
				t.Error(l)
			}
		}
	}
}
