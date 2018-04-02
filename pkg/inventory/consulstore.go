package inventory

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	consul "github.com/hashicorp/consul/api"
)

var DefaultConsulInventoryBase = "inventory"

func Marshal(src interface{}) ([]byte, error) {
	return json.Marshal(src)
}

func Unmarshal(src []byte, dst interface{}) error {
	return json.Unmarshal(src, &dst)
}

type ConsulStore struct {
	consul *consul.Client
	base   string
}

func NewConsulStore(client *consul.Client, base string) (*ConsulStore, error) {
	var c ConsulStore
	c.consul = client
	c.base = base
	return &c, nil
}

func (c *ConsulStore) InventoryObjectBase(obj interface{}) (string, error) {
	switch obj.(type) {
	case *types.InventoryNode:
		return fmt.Sprintf("%s/node", c.base), nil
	default:
		return "", fmt.Errorf("Object type not stored by this inventory store")
	}
}

func (c *ConsulStore) InventoryObjectKey(obj InventoryObject) (string, error) {
	base, err := c.InventoryObjectBase(obj)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", base, obj.ID()), nil
}

func (c *ConsulStore) Update(obj interface{}) error {
	var pair consul.KVPair
	var options consul.WriteOptions

	key, err := c.InventoryObjectKey(obj.(InventoryObject))
	if err != nil {
		return err
	}
	v, err := Marshal(obj)
	if err != nil {
		return err
	}
	pair.Key = key
	pair.Value = v

	_, err = c.consul.KV().Put(&pair, &options)
	return err
}

func (c *ConsulStore) Nodes() (map[string]*types.InventoryNode, error) {
	nodes := make(map[string]*types.InventoryNode)
	objbase, err := c.InventoryObjectBase(&types.InventoryNode{})
	if err != nil {
		return nil, err
	}
	pairs, _, err := c.consul.KV().List(objbase, &consul.QueryOptions{})
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nodes, nil
	}

	for _, pair := range pairs {
		node := &types.InventoryNode{}
		err = Unmarshal(pair.Value, node)
		if err != nil {
			log.Printf("Ignoring unparseable node %s: %v", pair.Key, err)
			continue
		}
		nodes[node.ID()] = node
	}
	return nodes, nil
}

func (c *ConsulStore) Delete(obj interface{}) error {
	key, err := c.InventoryObjectKey(obj.(InventoryObject))
	if err != nil {
		return err
	}
	_, err = c.consul.KV().Delete(key, &consul.WriteOptions{})
	return err
}

func (c *ConsulStore) Refresh() error {
	return nil
}
