package types

import (
	"fmt"
	"time"
)

type Node struct {
	InventoryID string
	*ChassisLocation
	ChassisSubIndex string
	Tags            []string
	Networks        map[string]*NICInfo
	Role            string
	Environment     string
	System          string
	Metadata        map[string]interface{}
	LastUpdated     time.Time
}

func NewNode() *Node {
	return &Node{ChassisLocation: &ChassisLocation{}}
}

func (n *Node) ID() string {
	return n.InventoryID
}

func (n *Node) Timestamp() int64 {
	return n.LastUpdated.Unix()
}

func (n *Node) Location() string {
	if n.ChassisLocation != nil && n.Rack != "" && n.ChassisSubIndex != "" {
		return fmt.Sprintf("%s-%0.2d-%s", n.Rack, n.BottomU, n.ChassisSubIndex)
	}

	if n.ChassisLocation != nil && n.Rack != "" {
		return fmt.Sprintf("%s-%0.2d", n.Rack, n.BottomU)
	}

	return ""
}

func (n *Node) Hostname() string {
	if location := n.Location(); location != "" && n.System != "" {
		return fmt.Sprintf("%s-%s", n.System, location)
	}

	if n.System != "" {
		return fmt.Sprintf("%s-%s", n.System, n.InventoryID)
	}

	return n.InventoryID
}
