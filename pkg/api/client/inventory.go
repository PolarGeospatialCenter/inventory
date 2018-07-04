package client

import (
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
)

type InventoryApi struct {
	AwsConfigs []*aws.Config
	BaseUrl    *url.URL
}

func NewInventoryApi(baseUrl *url.URL, configs ...*aws.Config) *InventoryApi {
	baseUrl, _ = baseUrl.Parse(baseUrl.Path + "/")
	return &InventoryApi{BaseUrl: baseUrl, AwsConfigs: configs}
}

func (i *InventoryApi) Url(endpointPath string) string {
	if endpointPath[0] == '/' {
		endpointPath = endpointPath[1:]
	}
	u, _ := i.BaseUrl.Parse(endpointPath)
	return u.String()
}

func (i *InventoryApi) Node() *Node {
	return &Node{Inventory: i}
}

func (i *InventoryApi) NodeConfig() *NodeConfig {
	return &NodeConfig{Inventory: i}
}

func (i *InventoryApi) Network() *Network {
	return &Network{Inventory: i}
}

func (i *InventoryApi) System() *System {
	return &System{Inventory: i}
}
