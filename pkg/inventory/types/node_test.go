package types

import (
	"encoding/json"
	"testing"
	"time"
)

func getTestNode() (*Node, string, string) {
	node := NewNode()
	node.InventoryID = "sample0001"
	node.ChassisLocation = &ChassisLocation{Building: "123 Fake St", Room: "305", Rack: "te12", BottomU: 4}
	node.ChassisSubIndex = "a"
	node.Tags = []string{"foo", "bar", "baz"}
	node.Networks = make(map[string]*NICInfo)
	nicInfo, _, _ := getTestNICInfo()
	node.Networks["test_phys"] = nicInfo
	node.Role = "worker"
	node.Environment = "production"
	node.System = "test"
	node.LastUpdated = time.Unix(123456789, 0).UTC()
	node.Metadata = make(map[string]interface{})
	node.Metadata["foo"] = "test"
	node.Metadata["bar"] = 34.1

	jsonString := `{"InventoryID":"sample0001","Building":"123 Fake St","Room":"305","Rack":"te12","BottomU":4,"ChassisSubIndex":"a","Tags":["foo","bar","baz"],"Networks":{"test_phys":{"MAC":"00:02:03:04:05:06","IP":"10.0.0.1"}},"Role":"worker","Environment":"production","System":"test","Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"}`
	yamlString := `inventoryid: sample0001
chassislocation:
  building: 123 Fake St
  room: "305"
  rack: te12
  bottomu: 4
chassissubindex: a
tags:
  - foo
  - bar
  - baz
networks:
  test_phys:
    mac: 00:02:03:04:05:06
    ip: 10.0.0.1
role: worker
environment: production
system: test
lastupdated: 1973-11-29T21:33:09Z
metadata:
  foo: test
  bar: 34.1
`
	return node, jsonString, yamlString
}

func TestNodeMarshalJSON(t *testing.T) {
	node, jsonstring, _ := getTestNode()
	actualString, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Unable to marshal: %v", err)
	}

	if string(actualString) != jsonstring {
		t.Fatalf("Got: %s, Expected: %s", string(actualString), jsonstring)
	}
}

func TestNodeUnmarshalJSON(t *testing.T) {
	expected, jsonString, _ := getTestNode()
	node := &Node{}
	testUnmarshalJSON(t, node, expected, jsonString)
}

func TestNodeUnmarshalYAML(t *testing.T) {
	expected, _, yamlString := getTestNode()
	node := &Node{}
	testUnmarshalYAML(t, node, expected, yamlString)
}

func TestNodeID(t *testing.T) {
	n := &Node{InventoryID: "foo2341"}
	if n.ID() != "foo2341" {
		t.Fatalf("Wrong node id returned: %s", n.ID())
	}
}

func TestLocationStringWithoutChassisSubIndex(t *testing.T) {
	l := &ChassisLocation{Building: "Fake St", Room: "123", Rack: "vf21", BottomU: 2}
	n := &Node{InventoryID: "foo2341", System: "bar", ChassisLocation: l}
	location := n.Location()
	if location != "vf21-02" {
		t.Fatalf("Incorrect location generated: %s", location)
	}
}

func TestLocationStringWithChassisSubIndex(t *testing.T) {
	l := &ChassisLocation{Building: "Fake St", Room: "123", Rack: "vf21", BottomU: 2}
	n := &Node{InventoryID: "foo2341", System: "bar", ChassisLocation: l, ChassisSubIndex: "a"}
	location := n.Location()
	if location != "vf21-02-a" {
		t.Fatalf("Incorrect location generated: %s", location)
	}
}

func TestHostnameWithChassisSubIndex(t *testing.T) {
	l := &ChassisLocation{Building: "Fake St", Room: "123", Rack: "vf21", BottomU: 2}
	n := &Node{InventoryID: "foo2341", System: "bar", ChassisLocation: l, ChassisSubIndex: "a"}
	hostname := n.Hostname()
	if hostname != "bar-vf21-02-a" {
		t.Fatalf("Incorrect hostname generated: %s", hostname)
	}
}

func TestHostnameWithoutChassisSubIndex(t *testing.T) {
	l := &ChassisLocation{Building: "Fake St", Room: "123", Rack: "vf21", BottomU: 2}
	n := &Node{InventoryID: "foo2341", System: "bar", ChassisLocation: l}
	hostname := n.Hostname()
	if hostname != "bar-vf21-02" {
		t.Fatalf("Incorrect hostname generated: %s", hostname)
	}
}

func TestHostnameWithoutChassisLocation(t *testing.T) {
	n := &Node{InventoryID: "foo2341", System: "bar", ChassisLocation: &ChassisLocation{}}
	hostname := n.Hostname()
	if hostname != "bar-foo2341" {
		t.Fatalf("Incorrect hostname generated: %s", hostname)
	}
}

func TestHostnameFailsafe(t *testing.T) {
	n := &Node{InventoryID: "foo2341"}
	hostname := n.Hostname()
	if hostname != "foo2341" {
		t.Fatalf("Incorrect hostname generated: %s", hostname)
	}
}

func TestNodeSetTimestamp(t *testing.T) {
	n := &Node{}
	ts := time.Now()
	n.SetTimestamp(ts)
	if n.Timestamp() != ts.Unix() {
		t.Errorf("Timestamp returned doesn't match the time set.")
	}
}
