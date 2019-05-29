package dynamodbclient

import (
	"fmt"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

type SystemStore struct {
	*DynamoDBStore
}

func (db *SystemStore) GetSystems() (map[string]*types.System, error) {
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

func (db *SystemStore) GetSystemByID(id string) (*types.System, error) {
	system := &types.System{}
	system.Name = id
	err := db.DynamoDBStore.get(system)
	return system, err
}

func (db *SystemStore) Exists(system *types.System) (bool, error) {
	return db.DynamoDBStore.exists(system)
}

func (db *SystemStore) Create(system *types.System) error {
	return db.DynamoDBStore.create(system)
}

func (db *SystemStore) Update(system *types.System) error {
	return db.DynamoDBStore.update(system)
}

func (db *SystemStore) Delete(system *types.System) error {
	return db.DynamoDBStore.delete(system)
}

func (db *SystemStore) ObjDelete(obj interface{}) error {
	system, ok := obj.(*types.System)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Delete(system)
}

func (db *SystemStore) ObjCreate(obj interface{}) error {
	system, ok := obj.(*types.System)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Create(system)
}

func (db *SystemStore) ObjUpdate(obj interface{}) error {
	system, ok := obj.(*types.System)
	if !ok {
		return ErrInvalidObjectType
	}
	return db.Update(system)
}

func (db *SystemStore) ObjExists(obj interface{}) (bool, error) {
	system, ok := obj.(*types.System)
	if !ok {
		return false, ErrInvalidObjectType
	}
	return db.Exists(system)
}
