package inventory

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"gopkg.in/yaml.v2"
)

type FileStore struct {
	path     string
	nodes    map[string]*types.Node
	networks map[string]*types.Network
	systems  map[string]*types.System
}

func NewFileStore(path string) (*FileStore, error) {
	return &FileStore{path: path, nodes: make(map[string]*types.Node), networks: make(map[string]*types.Network), systems: make(map[string]*types.System)}, nil
}

func (i *FileStore) Refresh() error {
	err := i.refreshNodes()
	if err != nil {
		return err
	}
	err = i.refreshNetworks()
	if err != nil {
		return err
	}
	err = i.refreshSystems()
	if err != nil {
		return err
	}
	return nil
}

func (i *FileStore) refreshNodes() error {
	nodepath := fmt.Sprintf("%s/node", i.path)
	err := filepath.Walk(nodepath, func(p string, info os.FileInfo, e error) error {
		if info.IsDir() {
			return nil
		}
		contents, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		n := types.NewNode()
		err = yaml.Unmarshal(contents, &n)
		if err != nil {
			return err
		}
		n.LastUpdated = info.ModTime()
		i.nodes[n.ID()] = n
		return nil
	})
	return err
}

func (i *FileStore) refreshNetworks() error {
	netpath := fmt.Sprintf("%s/network", i.path)
	err := filepath.Walk(netpath, func(p string, info os.FileInfo, e error) error {
		if info.IsDir() {
			return nil
		}
		contents, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		n := types.NewNetwork()
		err = yaml.Unmarshal(contents, &n)
		if err != nil {
			return err
		}
		n.LastUpdated = info.ModTime()
		i.networks[n.ID()] = n
		return nil
	})
	return err
}

func (i *FileStore) refreshSystems() error {
	syspath := fmt.Sprintf("%s/system", i.path)
	err := filepath.Walk(syspath, func(p string, info os.FileInfo, e error) error {
		if info.IsDir() {
			return nil
		}
		contents, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		s := types.NewSystem()
		err = yaml.Unmarshal(contents, &s)
		if err != nil {
			return err
		}
		s.LastUpdated = info.ModTime()
		i.systems[s.ID()] = s
		return nil
	})
	return err
}

func (i *FileStore) Nodes() (map[string]*types.InventoryNode, error) {
	err := i.Refresh()
	if err != nil {
		return nil, err
	}

	compiled := make(map[string]*types.InventoryNode)
	for _, n := range i.nodes {
		cnode, err := types.NewInventoryNode(n, types.NetworkMap(i.networks), types.SystemMap(i.systems))
		if err != nil {
			return nil, err
		}
		compiled[cnode.ID()] = cnode
	}
	return compiled, nil
}

func (i *FileStore) Update(obj interface{}) error {
	return fmt.Errorf("Not implemented")
}

func (i *FileStore) Delete(obj interface{}) error {
	return fmt.Errorf("Not implemented")
}
