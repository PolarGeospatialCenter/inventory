package client

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	gock "gopkg.in/h2non/gock.v1"
)

func TestNodeGetAll(t *testing.T) {
	gock.DisableNetworking()
	defer gock.EnableNetworking()
	defer gock.Off()
	testBaseUrl, _ := url.Parse("https://inventory.api.local/v0/")

	gock.New(testBaseUrl.String()).Get("node").Reply(http.StatusOK).BodyString(`[{"InventoryID": "test-000"}]`)

	nodes, err := NewInventoryApi(testBaseUrl, &aws.Config{Credentials: credentials.NewStaticCredentials("id", "secret", "token")}).Node().GetAll()
	if err != nil {
		t.Errorf("unable to get all nodes: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("got the wrong number of nodes: expected 1, got %d", len(nodes))
	}

	if nodes[0].InventoryID != "test-000" {
		t.Errorf("got wrong inventory id: %s", nodes[0].InventoryID)
	}
}
