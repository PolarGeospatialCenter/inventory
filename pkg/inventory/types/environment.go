package types

import "errors"

// ErrLogicalNetworkNotFound returned when a logical <-> physical network mapping
// doesn't exist
var ErrLogicalNetworkNotFound = errors.New("Network Not Found")

// Environment holds ipxe configuration as well as a mapping between logical and
// physical networks.
type Environment struct {
	IPXEUrl  string
	Networks map[string]string
	Metadata map[string]interface{} `json:",omitempty"`
}

// LookupLogicalNetworkName finds the logical network name associated with the
// physical network name provided
func (e *Environment) LookupLogicalNetworkName(physicalName string) (string, error) {
	for logical, physical := range e.Networks {
		if physical == physicalName {
			return logical, nil
		}
	}
	return "", ErrLogicalNetworkNotFound
}
