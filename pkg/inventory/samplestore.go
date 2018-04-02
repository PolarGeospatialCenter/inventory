package inventory

import (
	"net"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
)

// SampleInventoryStore is a pre populated sample inventory meant for testing
type SampleInventoryStore struct {
	*MemoryStore
}

// NewSampleInventoryStore returns a pre-populated sample InventoryStore
func NewSampleInventoryStore() (*SampleInventoryStore, error) {
	i := &SampleInventoryStore{MemoryStore: NewMemoryStore()}
	err := i.populate()
	return i, err
}

func (i *SampleInventoryStore) sampleNodes() []*types.Node {
	nodes := make([]*types.Node, 0)
	n1 := types.NewNode()
	n1.InventoryID = "sample0000"
	n1.ChassisLocation = &types.ChassisLocation{Building: "fakest", Room: "123", Rack: "ab00", BottomU: 2}
	n1.System = "bar"
	n1.Environment = "production"
	n1.Role = "worker"
	n1.Networks = make(map[string]*types.NICInfo)
	mac1, _ := net.ParseMAC("00-de-ad-be-ef-34")
	n1.Networks["prd_provisioning"] = &types.NICInfo{MAC: mac1, IP: net.ParseIP("127.0.0.1")}
	nodes = append(nodes, n1)
	n2 := types.NewNode()
	n2.InventoryID = "sample0001"
	n2.ChassisLocation = &types.ChassisLocation{Building: "fakest", Room: "123", Rack: "ab01", BottomU: 2}
	n2.System = "bar"
	n2.Environment = "test"
	n2.Role = "worker"
	n2.Networks = make(map[string]*types.NICInfo)
	mac2, _ := net.ParseMAC("00-de-ad-be-ef-35")
	n2.Networks["tst_provisioning"] = &types.NICInfo{MAC: mac2, IP: net.ParseIP("10.0.0.1")}
	nodes = append(nodes, n2)
	return nodes
}

func (i *SampleInventoryStore) sampleSystems() []*types.System {
	sys := make([]*types.System, 0)
	test := &types.Environment{IPXEUrl: "http://localhost/test", Networks: map[string]string{"provisioning": "tst_provisioning"}}
	prod := &types.Environment{IPXEUrl: "http://localhost/master", Networks: map[string]string{"provisioning": "prd_provisioning"}}
	sys = append(sys, &types.System{Name: "bar", ShortName: "bar", Environments: map[string]*types.Environment{"production": prod, "test": test}, Roles: []string{"worker", "test"}})
	return sys
}

func (i *SampleInventoryStore) sampleNetworks() error {
	err := i.Update(&types.Network{Name: "prd_provisioning", MTU: 1500})
	if err != nil {
		return err
	}
	err = i.Update(&types.Network{Name: "tst_provisioning", MTU: 1500})
	return err
}

func (i *SampleInventoryStore) populate() error {
	err := i.sampleNetworks()
	if err != nil {
		return err
	}

	for _, sys := range i.sampleSystems() {
		err := i.Update(sys)
		if err != nil {
			return err
		}
	}
	for _, node := range i.sampleNodes() {
		err := i.Update(node)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *SampleInventoryStore) Refresh() error {
	return nil
}
