package types

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func getTestNode() (*Node, string) {
	node := NewNode()
	node.InventoryID = "sample0002"
	node.ChassisLocation = &ChassisLocation{Building: "123 Fake St", Room: "305", Rack: "te12", BottomU: 4}
	node.ChassisSubIndex = "a"
	node.Tags = []string{"foo", "bar", "baz"}
	node.Networks = make(NICInfoMap)
	nicInfo, _ := getTestNICInfo()
	node.Networks["test_phys"] = nicInfo
	node.Role = "worker"
	node.Environment = "production"
	node.System = "test"
	node.LastUpdated = time.Unix(123456789, 0).UTC()
	node.Metadata = make(map[string]interface{})
	node.Metadata["foo"] = "test"
	node.Metadata["bar"] = 34.1

	jsonString := `{"InventoryID":"sample0002","Building":"123 Fake St","Room":"305","Rack":"te12","BottomU":4,"ChassisSubIndex":"a","Tags":["foo","bar","baz"],"Networks":{"test_phys":{"Metadata":{},"nics":["00:02:03:04:05:06"]}},"Role":"worker","Environment":"production","System":"test","Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"}`
	return node, jsonString
}

func TestNodeMarshalJSON(t *testing.T) {
	node, jsonstring := getTestNode()
	actualString, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Unable to marshal: %v", err)
	}

	if string(actualString) != jsonstring {
		t.Fatalf("Got: %s, Expected: %s", string(actualString), jsonstring)
	}
}

func TestNodeUnmarshalJSON(t *testing.T) {
	expected, jsonString := getTestNode()
	node := &Node{}
	testUnmarshalJSON(t, node, expected, jsonString)
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

func TestNodeNumericId(t *testing.T) {
	n := &Node{
		InventoryID: "test-0001-as-1",
	}

	id, err := n.NumericId()

	if id != 11 {
		t.Errorf("Returned id did not match expected: got %d expected 11", id)
	}

	if err != nil {
		t.Errorf("Got an error when we shouldn't have: %v", err)
	}
}

func TestNodeNumericIdNil(t *testing.T) {
	n := &Node{
		InventoryID: "test-as",
	}

	_, err := n.NumericId()

	if err, ok := err.(*strconv.NumError); !ok {
		t.Errorf("Did not get the error we should have: got %v", err)
	}
}
