package inventory

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func TestGitStoreInventory(t *testing.T) {
	path, _ := os.Getwd()
	testdir := filepath.Join(path, "..", "..", "test", "data", "gitstore_0")
	t.Logf("Loading test gitrepo from: %s", testdir)
	cloneOpts := &git.CloneOptions{
		URL: fmt.Sprintf("file://%s", testdir),
	}
	repo, err := git.Clone(memory.NewStorage(), nil, cloneOpts)
	if err != nil {
		t.Fatalf("Unable to clone repo: %v", err)
	}

	store := NewGitStore(repo, &git.FetchOptions{}, "master")
	store.Refresh()
	nodes, err := store.Nodes()
	if err != nil {
		t.Errorf("Unable to get nodes from git inventory: %v", err)
	}

	if len(nodes) != 3 {
		t.Errorf("Wrong number of nodes returned: %d expecting 3", len(nodes))
	}

	lastUpdates := map[string]string{
		"node0001": "2018-01-17T19:29:43Z",
		"node0002": "2018-01-17T19:28:44Z",
		"node0003": "2018-01-17T19:28:44Z",
	}

	for nodeID, timestamp := range lastUpdates {
		expectedTime := &time.Time{}
		err := expectedTime.UnmarshalText([]byte(timestamp))
		if err != nil {
			t.Errorf("Unable to parse expected time '%s': %v", timestamp, err)
		}
		actualTime := nodes[nodeID].LastUpdated
		if actualTime.UnixNano() != expectedTime.UnixNano() {
			t.Errorf("Wrong last updated time for %s: actual %s, expected %s", nodeID, actualTime, expectedTime)
		}
	}

	path, _ = os.Getwd()
	testdir = filepath.Join(path, "..", "..", "test", "data", "gitstore_1")
	t.Logf("Resetting git remote for fetch test")
	remote, err := store.repo.Remote("origin")
	if err != nil {
		t.Errorf("Unable to get remote config for origin: %v", err)
	}
	remote.Config().URLs = []string{fmt.Sprintf("file://%s", testdir)}

	err = store.Refresh()
	if err != nil {
		t.Errorf("Unable to refresh from gitstore_1: %v", err)
	}

	nodes, err = store.Nodes()
	if err != nil {
		t.Errorf("Unable to get nodes from git inventory: %v", err)
	}

	if len(nodes) != 3 {
		t.Errorf("Wrong number of nodes returned: %d expecting 3", len(nodes))
	}

	lastUpdates = map[string]string{
		"node0001": "2018-01-17T19:29:43Z",
		"node0002": "2018-01-17T19:28:44Z",
		"node0003": "2018-01-17T19:31:44Z",
	}

	for nodeID, timestamp := range lastUpdates {
		expectedTime := &time.Time{}
		err := expectedTime.UnmarshalText([]byte(timestamp))
		if err != nil {
			t.Errorf("Unable to parse expected time '%s': %v", timestamp, err)
		}
		actualTime := nodes[nodeID].LastUpdated
		if actualTime.UnixNano() != expectedTime.UnixNano() {
			t.Errorf("Wrong last updated time for %s: actual %s, expected %s", nodeID, actualTime, expectedTime)
		}
	}

}
