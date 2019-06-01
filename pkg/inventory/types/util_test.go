package types

import (
	"encoding/json"
	"testing"

	"github.com/go-test/deep"
	yaml "gopkg.in/yaml.v2"
)

func testUnmarshalJSON(t *testing.T, dst interface{}, expected interface{}, jsonString string) {
	err := json.Unmarshal([]byte(jsonString), dst)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML: %v", err)
	}

	if diff := deep.Equal(dst, expected); diff != nil {
		t.Errorf("Unmarshaled not equal to expected:")
		for _, d := range diff {
			t.Error(d)
		}
	}
	t.Logf("Got: %v", jsonString)
	t.Logf("Unmarshaled: %v", dst)
}

func testUnmarshalYAML(t *testing.T, dst interface{}, expected interface{}, yamlString string) {
	err := yaml.Unmarshal([]byte(yamlString), dst)
	if err != nil {
		t.Fatalf("Unable to unmarshal: %v", err)
	}

	if diff := deep.Equal(dst, expected); diff != nil {
		t.Errorf("Unmarshaled not equal to expected:")
		for _, d := range diff {
			t.Error(d)
		}
		t.FailNow()
	}
}
