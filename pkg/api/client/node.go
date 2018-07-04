package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

type Node struct {
	Inventory *InventoryApi
}

func (n *Node) GetAll() ([]*types.Node, error) {
	client := NewRestClient(n.Inventory.AwsConfigs...)

	response, err := client.Client().NewRequest().Execute(http.MethodGet, n.Inventory.Url("/node"))
	if err != nil {
		return nil, fmt.Errorf("unable to get nodes: %v", err)
	}
	nodes := []*types.Node{}
	err = json.Unmarshal(response.Body(), &nodes)
	return nodes, err
}
