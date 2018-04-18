package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/google/go-github/github"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

type config struct {
	deployKey []byte
	branch    string
}

func loadConfig(ctx context.Context) (config, error) {
	lctx := lambdacontext.FromContext(ctx)
	prefix := arnToParameterPrefix(lctx.InvokedFunctionArn)

	m := ssm.New(session.New())
	m.GetParameter()
	return config{deployKey: []byte("")}, nil
}

func arnToParameterPrefix(arn string) (string, error) {
	return arn
}

func parseEventRecord(record events.SNSEventRecord) (*github.PushEvent, error) {
	payload := &github.PushEvent{}
	err := json.Unmarshal([]byte(record.SNS.Message), payload)
	return payload, err
}

func cloneGitRepo(path string, deployKey []byte, sshRepoURL string, branch string) (*git.Repository, error) {
	cloneOptions := &git.CloneOptions{URL: sshRepoURL}
	auth, err := ssh.NewPublicKeys("git", deployKey, "")
	if err != nil {
		return nil, fmt.Errorf("unable to load deploy key: %v", err)
	}
	cloneOptions.Auth = auth

	if _, err = os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0700)
	}

	return git.PlainClone(path, false, cloneOptions)
}

// Handler handles sns events representing github events.
func Handler(ctx context.Context, snsEvent events.SNSEvent) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		log.Printf("unable to load config: %v", err)
		return
	}

	for _, record := range snsEvent.Records {
		// Process push events
		event, err := parseEventRecord(record)
		if err != nil {
			// TODO: handle errors
			log.Printf("Error processing event: %v", err)
			continue
		}

		// Create GitStore
		path := "/tmp/repo"
		deployKey := cfg.deployKey
		branch := cfg.branch
		repo, err := cloneGitRepo(path, deployKey, event.GetRepo().GetSSHURL(), branch)
		if err != nil {
			log.Printf("Error cloning git repo: %v", err)
			continue
		}
		gitStore := inventory.NewGitStore(repo, &git.FetchOptions{}, branch)

		// Create DynamoDBStore
		// TODO: Get DynamoDB table map arns from somewhere
		db := dynamodb.New(session.New())
		ddbStore := inventory.NewDynamoDBStore(db, nil)

		// update dynamodb from git
		err = ddbStore.UpdateFromInventoryStore(gitStore)
		if err != nil {
			log.Printf("Error updating inventory store: %v", err)
			continue
		}
	}
}

func main() {
	lambda.Start(Handler)
}
