package inventory

import (
	"errors"
	"fmt"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

var (
	ErrNodeNotFound   = errors.New("Node not found")
	ErrObjectNotFound = errors.New("Object not found")
)

type InventoryObject interface {
	ID() string
	Timestamp() int64
	SetTimestamp(time.Time)
}

type InventoryStore interface {
	Nodes() (map[string]*types.InventoryNode, error)
	Refresh() error
	Update(interface{}) error
	Delete(interface{}) error
}

type Inventory struct {
	store InventoryStore
}

func NewInventory(store InventoryStore) (*Inventory, error) {
	return &Inventory{store: store}, nil
}

// GetNodeByHostname returns the first node found in inventory whose short hostname
// matches the supplied name
func (i *Inventory) GetNodeByHostname(name string) (*types.InventoryNode, error) {
	nodes, err := i.store.Nodes()
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		if node.Hostname == name {
			return node, nil
		}
	}
	return nil, ErrNodeNotFound
}

func (i *Inventory) GetNode(id string) (*types.InventoryNode, error) {
	nodes, err := i.store.Nodes()
	if err != nil {
		return nil, err
	}
	node, ok := nodes[id]
	if !ok {
		return nil, fmt.Errorf("Node not found")
	}
	return node, nil
}

func (i *Inventory) CopyUpdatedNodes(store InventoryStore) error {
	myNodes, err := i.store.Nodes()
	if err != nil {
		return err
	}

	otherNodes, err := store.Nodes()
	if err != nil {
		return err
	}

	for id, node := range myNodes {
		other, ok := otherNodes[id]
		if !ok || node.LastUpdated.Unix() > other.LastUpdated.Unix() {
			err = store.Update(node)
			if err != nil {
				return err
			}
		}
	}

	for id, node := range otherNodes {
		if _, ok := myNodes[id]; !ok {
			err = store.Delete(node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
