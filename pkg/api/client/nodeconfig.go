package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

type NodeConfig struct {
	Inventory *InventoryApi
}

func (c *NodeConfig) GetAll() ([]*types.InventoryNode, error) {
	client := NewRestClient(c.Inventory.AwsConfigs...)
	response, err := client.Client().NewRequest().Execute(http.MethodGet, c.Inventory.Url("/nodeconfig"))
	if err != nil {
		return nil, fmt.Errorf("unable to get nodes: %v", err)
	}
	nodes := []*types.InventoryNode{}
	err = json.Unmarshal(response.Body(), &nodes)
	return nodes, err
}
