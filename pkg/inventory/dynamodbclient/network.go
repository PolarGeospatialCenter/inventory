package dynamodbclient

import (
	"fmt"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

type NetworkStore struct {
	*DynamoDBStore
}

func (db *NetworkStore) GetNetworks() (map[string]*types.Network, error) {
	networkList := make([]*types.Network, 0, 0)
	err := db.getAll(&networkList)
	if err != nil {
		return nil, fmt.Errorf("error getting all networks: %v", err)
	}
	networks := make(map[string]*types.Network)
	for _, n := range networkList {
		networks[n.ID()] = n
	}
	return networks, nil
}

func (db *NetworkStore) GetNetworkByID(id string) (*types.Network, error) {
	network := &types.Network{}
	network.Name = id
	err := db.get(network)
	if err != nil {
		return nil, err
	}

	if network.Subnets == nil {
		network.Subnets = make([]*types.Subnet, 0)
	}
	return network, err
}

func (db *NetworkStore) Exists(network *types.Network) (bool, error) {
	return db.DynamoDBStore.exists(network)
}

func (db *NetworkStore) Create(network *types.Network) error {
	return db.DynamoDBStore.create(network)
}

func (db *NetworkStore) Update(network *types.Network) error {
	return db.DynamoDBStore.update(network)

}

func (db *NetworkStore) Delete(network *types.Network) error {
	return db.DynamoDBStore.delete(network)
}

func (db *NetworkStore) ObjDelete(obj interface{}) error {
	network, ok := obj.(*types.Network)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Delete(network)
}

func (db *NetworkStore) ObjCreate(obj interface{}) error {
	network, ok := obj.(*types.Network)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Create(network)
}

func (db *NetworkStore) ObjUpdate(obj interface{}) error {
	network, ok := obj.(*types.Network)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Update(network)
}

func (db *NetworkStore) ObjExists(obj interface{}) (bool, error) {
	network, ok := obj.(*types.Network)
	if !ok {
		return false, ErrInvalidObjectType
	}
	return db.Exists(network)
}
