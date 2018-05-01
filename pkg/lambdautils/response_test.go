package lambdautils

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-test/deep"
)

func TestNewJSONAPIGatewayProxyResponseString(t *testing.T) {
	type testCase struct {
		Status       int
		Headers      map[string]string
		BodyData     interface{}
		ExpectedBody string
	}

	cases := []testCase{
		testCase{http.StatusOK, map[string]string{}, "Hello World!", `"Hello World!"`},
		testCase{http.StatusOK, map[string]string{}, []string{"Hello World!"}, `["Hello World!"]`},
		testCase{http.StatusInternalServerError, map[string]string{}, fmt.Errorf("Error string"), `{"status":"Internal Server Error","error":"Error string"}`},
		testCase{http.StatusOK, map[string]string{}, struct {
			Test string `json:"test"`
		}{Test: "Hello World!"}, `{"test":"Hello World!"}`},
	}

	for _, c := range cases {
		response, err := NewJSONAPIGatewayProxyResponse(c.Status, c.Headers, c.BodyData)
		if err != nil {
			t.Fatalf("Unable to create response object from string: %v", err)
		}

		if response.StatusCode != c.Status {
			t.Errorf("Wrong status code returned: got %d; expected %d", response.StatusCode, c.Status)
		}

		if diff := deep.Equal(response.Headers, c.Headers); len(diff) > 0 {
			t.Error("header mismatch:")
			for _, l := range diff {
				t.Error(l)
			}
		}

		if response.Body != c.ExpectedBody {
			t.Errorf("Wrong body returned: got '%s'; expected '%s'", response.Body, c.ExpectedBody)
		}
	}
}
