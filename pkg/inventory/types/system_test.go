package types

import (
	"encoding/json"
	"testing"
	"time"
)

func getTestSystem() (*System, string, string) {
	sys := NewSystem()
	sys.Name = "Test System"
	sys.ShortName = "test"
	sys.Roles = []string{"worker"}
	sys.Environments = make(map[string]*Environment)
	sys.Environments["production"] = &Environment{IPXEUrl: "http://localhost:8080/", Networks: map[string]string{"logical": "test_phys"}, Metadata: map[string]interface{}{"key": "value"}}
	sys.LastUpdated = time.Unix(123456789, 0).UTC()
	sys.Metadata = make(map[string]interface{})
	sys.Metadata["foo"] = "test"
	sys.Metadata["bar"] = 34.1

	jsonString := `{"Name":"Test System","ShortName":"test","Environments":{"production":{"IPXEUrl":"http://localhost:8080/","Networks":{"logical":"test_phys"},"Metadata":{"key":"value"}}},"Roles":["worker"],"Metadata":{"bar":34.1,"foo":"test"},"LastUpdated":"1973-11-29T21:33:09Z"}`
	yamlString := `name: Test System
shortname: test
roles:
  - worker
environments:
  production:
    ipxeurl: "http://localhost:8080/"
    networks:
      logical: test_phys
    metadata:
      key: value
lastupdated: 1973-11-29T21:33:09Z
metadata:
  foo: test
  bar: 34.1
`
	return sys, jsonString, yamlString
}

func TestSystemMarshalJSON(t *testing.T) {
	sys, jsonstring, _ := getTestSystem()
	actualString, err := json.Marshal(sys)
	if err != nil {
		t.Fatalf("Unable to marshal: %v", err)
	}

	if string(actualString) != jsonstring {
		t.Fatalf("Got: %s, Expected: %s", string(actualString), jsonstring)
	}
}

func TestSystemUnmarshalJSON(t *testing.T) {
	expected, jsonString, _ := getTestSystem()
	sys := &System{}
	testUnmarshalJSON(t, sys, expected, jsonString)
}

func TestSystemUnmarshalYAML(t *testing.T) {
	expected, _, yamlString := getTestSystem()
	sys := &System{}
	testUnmarshalYAML(t, sys, expected, yamlString)
}

func TestSystemID(t *testing.T) {
	sys := &System{Name: "Test System", ShortName: "test"}
	if sys.ID() != "test" {
		t.Fatalf("Wrong system ID returned: %s", sys.ID())
	}

	sys = &System{Name: "Test System"}
	if sys.ID() != "Test System" {
		t.Fatalf("Wrong system ID returned: %s", sys.ID())
	}
}
