package types

import (
	"encoding/json"
	"fmt"
	"testing"
)

func getTestEnvironment() (*Environment, string) {
	networks := make(map[string]string)
	networks["foo"] = "prd_foo"
	networks["bar"] = "prd_bar"
	env := &Environment{IPXEUrl: "http://localhost:8080/test", Networks: networks, Metadata: map[string]interface{}{"key": "value"}}
	marshaled := fmt.Sprintf(`{"IPXEUrl":"%s","Networks":{"bar":"prd_bar","foo":"prd_foo"},"Metadata":{"key":"value"}}`, env.IPXEUrl)
	return env, marshaled
}

func TestLookupLogicalNetworkNameOK(t *testing.T) {
	env, _ := getTestEnvironment()
	logicalNet, err := env.LookupLogicalNetworkName("prd_foo")
	if err != nil {
		t.Fatalf("Unable to lookup logical network for prd_foo: %v", err)
	}
	if logicalNet != "foo" {
		t.Fatalf("Wrong logical network returned: %v", logicalNet)
	}
}

func TestLookupLogicalNetworkNameFailed(t *testing.T) {
	env, _ := getTestEnvironment()
	_, err := env.LookupLogicalNetworkName("non_exist")
	if err != ErrLogicalNetworkNotFound {
		t.Fatalf("Incorrect error returned for non existent network lookup: %v", err)
	}
}

func TestEnvironmentMarshal(t *testing.T) {
	env, expected := getTestEnvironment()
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Unable to marshal environment: %v", err)
	}

	if string(b) != expected {
		t.Fatalf("Wrong marshaled text returned: '%s', Expected: '%s' ", string(b), expected)
	}
}

func TestEnvironmentUnmarshal(t *testing.T) {
	expected, marshaled := getTestEnvironment()
	env := &Environment{}
	testUnmarshalJSON(t, env, expected, marshaled)
}
