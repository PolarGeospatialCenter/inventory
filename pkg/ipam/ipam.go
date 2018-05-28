package ipam

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/azenk/iputils"
)

var (
	ErrAllocationNotImplemented = errors.New("ipv4 allocation not implemented")
)

func IsV6(ip net.IP) bool {
	return ip.To4() == nil && ip.To16() != nil
}

func LocationBits(location *inventorytypes.ChassisLocation, sublocation string) (uint64, error) {
	rackInt, err := strconv.ParseUint(location.Rack[len(location.Rack)-4:], 36, 32)
	if err != nil {
		return 0, err
	}

	subChassisInt := uint64(0)
	if sublocation != "" {
		subChassisInt, err = strconv.ParseUint(sublocation, 16, 32)
		if err != nil {
			return 0, err
		}
	}

	locationBits := rackInt << 10
	locationBits |= (uint64(location.BottomU) << 4) & 0x03f0
	locationBits |= subChassisInt & 0x0f
	return locationBits, nil
}

func GetIPByLocation(subnet *inventorytypes.Subnet, location *inventorytypes.ChassisLocation, sublocation string) (net.IP, error) {

	if IsV6(subnet.Cidr.IP) {
		locationBits, err := LocationBits(location, sublocation)
		if err != nil {
			return net.IP{}, fmt.Errorf("unable to calculate location bits: %v", err)
		}
		// flip msb to indicate that this is a host ip, not the host prefix
		locationBits |= 1 << 31

		startoffset, _ := subnet.Cidr.Mask.Size()

		newIp, err := iputils.SetBits(subnet.Cidr.IP, locationBits, uint(startoffset), 32)
		if err != nil {
			return net.IP{}, fmt.Errorf("unable to set location bits on subnet: %v", err)
		}

		newIp, err = iputils.SetBits(newIp, 1, uint(startoffset+32), uint(128-(startoffset+32)))
		if err != nil {
			return net.IP{}, fmt.Errorf("unable to set host bits on subnet: %v", err)
		}

		return newIp, nil
	} else {
		return net.IP{}, ErrAllocationNotImplemented
	}
}

func GetRangeByLocation(subnet *inventorytypes.Subnet, location *inventorytypes.ChassisLocation, sublocation string) (net.IP, net.IP, error) {

	if IsV6(subnet.Cidr.IP) {
		locationBits, err := LocationBits(location, sublocation)
		if err != nil {
			return net.IP{}, net.IP{}, fmt.Errorf("unable to calculate location bits: %v", err)
		}

		startoffset, _ := subnet.Cidr.Mask.Size()

		newIp, err := iputils.SetBits(subnet.Cidr.IP, locationBits, uint(startoffset), 32)
		if err != nil {
			return net.IP{}, net.IP{}, fmt.Errorf("unable to set location bits on subnet: %v", err)
		}

		newIpEnd, err := iputils.SetBits(newIp, ^uint64(0), uint(startoffset+32), uint(128-(startoffset+32)))
		if err != nil {
			return net.IP{}, net.IP{}, fmt.Errorf("unable to set end of range bits on subnet: %v", err)
		}

		return newIp, newIpEnd, nil
	} else {
		return net.IP{}, net.IP{}, ErrAllocationNotImplemented
	}
}
