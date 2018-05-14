package main

import (
	"log"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	git "gopkg.in/src-d/go-git.v4"
)

func main() {
	pathToRepo := "."
	branch := "master"
	aws_profile := "pgc-dev"
	aws_region := "us-east-2"

	// open git repo
	repo, err := git.PlainOpen(pathToRepo)
	if err != nil {
		log.Fatalf("Unable to open git repo: %v", err)
	}

	head, err := repo.Head()
	log.Printf("Repo Head: %v, Err: %v", head, err)

	gitStore := inventory.NewGitStore(repo, &git.FetchOptions{}, branch)
	err = gitStore.Refresh()
	if err != nil {
		log.Fatalf("Unable to refresh state of git repo: %v", err)
	}
	nodes, err := gitStore.GetNodes()
	if err != nil {
		log.Fatalf("Unable to get nodes from git: %v", err)
	}
	log.Printf("Found %d nodes: %v", len(nodes), nodes)

	// load aws credentials and connect to dynamodb
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: aws_profile,
		Config:  aws.Config{Region: aws.String(aws_region)},
	})
	if err != nil {
		log.Fatalf("Unable to load aws credentials: %v", err)
	}

	db := dynamodb.New(sess)
	ddbStore := inventory.NewDynamoDBStore(db, nil)

	err = ddbStore.UpdateFromInventoryStore(gitStore)
	if err != nil {
		log.Fatalf("Error updating dynamodb from git store: %v", err)
	}
}
