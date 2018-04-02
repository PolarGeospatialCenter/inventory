package inventory

import (
	"fmt"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

type MemoryStore struct {
	nodes          map[string]*types.Node
	networks       map[string]*types.Network
	systems        map[string]*types.System
	inventorynodes map[string]*types.InventoryNode
}

func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{}
	store.nodes = make(map[string]*types.Node)
	store.networks = make(map[string]*types.Network)
	store.systems = make(map[string]*types.System)
	store.inventorynodes = make(map[string]*types.InventoryNode)
	return store
}

func (m *MemoryStore) Nodes() (map[string]*types.InventoryNode, error) {
	err := m.rebuildInventory()
	if err != nil {
		return nil, err
	}
	return m.inventorynodes, nil
}

func (m *MemoryStore) rebuildInventory() error {
	for _, node := range m.nodes {
		inode, err := types.NewInventoryNode(node, types.NetworkMap(m.networks), types.SystemMap(m.systems))
		if err != nil {
			return err
		}
		m.inventorynodes[inode.ID()] = inode
	}
	return nil
}

func (m *MemoryStore) Update(obj interface{}) error {
	switch obj.(type) {
	case *types.Node:
		m.nodes[obj.(*types.Node).ID()] = obj.(*types.Node)
	case *types.Network:
		m.networks[obj.(*types.Network).ID()] = obj.(*types.Network)
	case *types.System:
		m.systems[obj.(*types.System).ID()] = obj.(*types.System)
	case *types.InventoryNode:
		m.inventorynodes[obj.(*types.InventoryNode).ID()] = obj.(*types.InventoryNode)
	default:
		return fmt.Errorf("Type not supported by this data store")
	}
	return nil
}

func (m *MemoryStore) Delete(obj interface{}) error {
	switch obj.(type) {
	case *types.Node:
		delete(m.nodes, obj.(*types.Node).ID())
	case *types.Network:
		delete(m.networks, obj.(*types.Network).ID())
	case *types.System:
		delete(m.systems, obj.(*types.System).ID())
	case *types.InventoryNode:
		delete(m.inventorynodes, obj.(*types.System).ID())
	default:
		return fmt.Errorf("Type not supported by this data store")
	}
	return nil
}

func (m *MemoryStore) Refresh() error {
	return nil
}
