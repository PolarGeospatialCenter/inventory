package ipam

import (
	"net"
	"testing"

	"github.com/go-test/deep"
)

func TestGetIpByIdV4(t *testing.T) {
	_, subnet, _ := net.ParseCIDR("10.1.0.0/16")

	reservedAdress := net.ParseIP("10.1.3.254")
	id := 20000

	ip, err := GetIpById(id, subnet, reservedAdress)

	if err != nil {
		t.Errorf("Got error when we should not have: %v", err)
	}

	expectedIp := net.ParseIP("10.1.78.32")
	if !ip.Equal(expectedIp) {
		t.Errorf("Ip returned did not match what we wanted: got %s expected %s", ip.String(), expectedIp.String())
	}
}

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
		Subnet          *net.IPNet
		Rack            string
		BottomU         uint
		ChassisSubIndex string
		ExpectedIp      net.IP
		ExpectedErr     error
	}

	_, v6cidrA, _ := net.ParseCIDR("2001:db8::/64")
	_, v6cidrB, _ := net.ParseCIDR("2001:db8::/56")
	_, v4cidr, _ := net.ParseCIDR("10.0.0.0/24")

	cases := []*testCase{
		&testCase{
			Subnet:          v6cidrB,
			Rack:            "xr20",
			BottomU:         31,
			ChassisSubIndex: "a",
			ExpectedIp:      net.ParseIP("2001:db8::e0:1ce1:fa00:0:1"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          v6cidrA,
			Rack:            "xr20",
			BottomU:         31,
			ChassisSubIndex: "a",
			ExpectedIp:      net.ParseIP("2001:db8::e01c:e1fa:0:1"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          v4cidr,
			Rack:            "xr20",
			BottomU:         31,
			ChassisSubIndex: "a",
			ExpectedIp:      net.IP{},
			ExpectedErr:     ErrAllocationNotImplemented,
		},
	}

	for _, c := range cases {
		ip, err := GetIPByLocation(c.Subnet, c.Rack, c.BottomU, c.ChassisSubIndex)
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
		Subnet          *net.IPNet
		Rack            string
		BottomU         uint
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
			Subnet:          v6cidrB,
			Rack:            "xr20",
			BottomU:         31,
			ChassisSubIndex: "a",
			ExpectedStartIp: net.ParseIP("2001:db8:0:60:1ce1:fa00::"),
			ExpectedEndIp:   net.ParseIP("2001:db8:0:60:1ce1:faff:ffff:ffff"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          v6cidrA,
			Rack:            "xr20",
			BottomU:         31,
			ChassisSubIndex: "a",
			ExpectedStartIp: net.ParseIP("2001:db8:0:0:601c:e1fa:0::"),
			ExpectedEndIp:   net.ParseIP("2001:db8::601c:e1fa:ffff:ffff"),
			ExpectedErr:     nil,
		},
		&testCase{
			Subnet:          v4cidr,
			Rack:            "xr20",
			BottomU:         31,
			ChassisSubIndex: "a",
			ExpectedStartIp: net.IP{},
			ExpectedEndIp:   net.IP{},
			ExpectedErr:     ErrAllocationNotImplemented,
		},
	}

	for _, c := range cases {
		t.Logf("Testing %s: %s", c.Subnet.String(), c.ExpectedStartIp)
		ipStart, ipEnd, err := GetRangeByLocation(c.Subnet, c.Rack, c.BottomU, c.ChassisSubIndex)
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
