package dynamodbclient

import (
	"fmt"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

func (db *DynamoDBStore) GetSystems() (map[string]*types.System, error) {
	systemList := make([]*types.System, 0, 0)
	err := db.getAll(&systemList)
	if err != nil {
		return nil, fmt.Errorf("error getting all systems: %v", err)
	}
	systems := make(map[string]*types.System)
	for _, s := range systemList {
		systems[s.ID()] = s
	}
	return systems, nil
}

func (db *DynamoDBStore) GetSystemByID(id string) (*types.System, error) {
	system := &types.System{}
	system.Name = id
	err := db.Get(system)
	return system, err
}
