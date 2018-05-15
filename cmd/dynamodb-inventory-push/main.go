package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	git "gopkg.in/src-d/go-git.v4"
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "This program uploads the contents of an Inventory GIT repo to a DynamoDB instance.\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}

	pathToRepo := flag.String("git_path", ".", "The path to the git repo.")
	branch := flag.String("git_branch", "master", "The branch of the git repo.")
	aws_profile := flag.String("aws_profile", "default", "The AWS profile to use.")
	aws_region := flag.String("aws_region", "us-east-2", "The AWS region to use.")
	flag.Parse()

	// open git repo
	repo, err := git.PlainOpen(*pathToRepo)
	if err != nil {
		log.Fatalf("Unable to open git repo: %v", err)
	}

	head, err := repo.Head()
	log.Printf("Repo Head: %v, Err: %v", head, err)

	gitStore := inventory.NewGitStore(repo, &git.FetchOptions{}, *branch)
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
		Profile: *aws_profile,
		Config:  aws.Config{Region: aws.String(*aws_region)},
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
