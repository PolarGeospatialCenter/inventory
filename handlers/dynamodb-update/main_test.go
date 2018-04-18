package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestParsePayload(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working directory: %v", err)
	}

	testPayloadFile := path.Join(cwd, "..", "..", "test", "github_hooks", "push_payload.json")
	data, err := ioutil.ReadFile(testPayloadFile)
	if err != nil {
		t.Fatalf("Unable to read test payload from file: %v", err)
	}

	record := events.SNSEventRecord{SNS: events.SNSEntity{Message: string(data)}}
	payload, err := parseEventRecord(record)
	if err != nil {
		t.Errorf("Unable to parse event record: %v", err)
	}

	if payload.GetRepo().GetSSHURL() != "git@github.example.tld:ou-foo/repo-bar.git" {
		t.Errorf("Repo ssh url doesn't match expected: %s", payload.GetRepo().GetSSHURL())
	}
}
