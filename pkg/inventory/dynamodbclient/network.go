package dynamodbclient

import (
	"fmt"
	"log"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

func (db *DynamoDBStore) GetNetworks() (map[string]*types.Network, error) {
	networkList := make([]*types.Network, 0, 0)
	err := db.getAll(&networkList)
	if err != nil {
		return nil, fmt.Errorf("error getting all networks: %v", err)
	}
	log.Printf("Network List returned: %v", networkList)
	networks := make(map[string]*types.Network)
	for _, n := range networkList {
		networks[n.ID()] = n
	}
	return networks, nil
}

func (db *DynamoDBStore) GetNetworkByID(id string) (*types.Network, error) {
	network := &types.Network{}
	network.Name = id
	err := db.Get(network)
	if err != nil {
		return nil, err
	}

	if network.Subnets == nil {
		network.Subnets = make([]*types.Subnet, 0)
	}
	return network, err
}
