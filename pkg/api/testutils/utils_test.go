package testutils

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalAndCompare(t *testing.T) {
	obj := map[string]string{
		"test": "foo",
	}

	marshaled, err := json.Marshal(obj)
	if err != nil {
		t.Errorf("error marshaling test: %v", err)
	}

	diff := UnmarshalAndCompare(string(marshaled), obj)
	if len(diff) > 0 {
		t.Errorf("Compare returned diff when it shouldn't have:")
		for _, l := range diff {
			t.Error(l)
		}
	}

	obj["test2"] = "value"

	diff = UnmarshalAndCompare(string(marshaled), obj)
	if len(diff) == 0 {
		t.Errorf("Compare returned no diff when it should have")
	}
}
