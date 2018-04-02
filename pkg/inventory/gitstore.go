package inventory

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	git "gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/yaml.v2"
)

type FileUpdateTimes map[string]time.Time

func NewFileUpdateTimes() FileUpdateTimes {
	return FileUpdateTimes{}
}

// Updates the most recent update time if updateTime is greater than the last known.
// returns true if updated, false otherwise
func (t FileUpdateTimes) Update(name string, updateTime time.Time) bool {
	lastUpdatedTime, ok := t[name]
	if !ok || lastUpdatedTime.Unix() < updateTime.Unix() {
		t[name] = updateTime
		return true
	}
	return false
}

type GitStore struct {
	repo         *git.Repository
	fetchOptions *git.FetchOptions
	branch       string
	cache        *MemoryStore
}

func NewGitStore(repo *git.Repository, fetchOptions *git.FetchOptions, branch string) *GitStore {
	return &GitStore{repo: repo, branch: branch, cache: NewMemoryStore(), fetchOptions: fetchOptions}
}

func (g *GitStore) Nodes() (map[string]*types.InventoryNode, error) {
	return g.cache.Nodes()
}

func (g *GitStore) refString() gitplumbing.ReferenceName {
	return gitplumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", g.branch))
}

func (g *GitStore) Fetch(options *git.FetchOptions) error {
	log.Println("Fetching updates")
	err := g.repo.Fetch(options)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	log.Println("Wiping cache")
	g.cache = NewMemoryStore()

	log.Println("Getting head of branch")
	head, err := g.repo.Reference(g.refString(), true)
	if err != nil {
		return err
	}

	log.Println("Get commit associated with head")
	commit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return err
	}

	log.Println("Get tree object for: %s", commit)
	tree, err := g.repo.TreeObject(commit.TreeHash)
	if err != nil {
		return err
	}

	updateTimes, err := g.fileUpdateTimes()
	if err != nil {
		return err
	}

	log.Println("Walk tree")
	err = tree.Files().ForEach(func(f *gitobject.File) error {
		return g.LoadObject(f, updateTimes[f.Name])
	})
	log.Println("Update finished")
	return err
}

func (g *GitStore) fileUpdateTimes() (FileUpdateTimes, error) {
	updateTimes := NewFileUpdateTimes()

	head, err := g.repo.Reference(g.refString(), true)
	if err != nil {
		return nil, err
	}

	headCommit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}

	pTime := headCommit.Author.When
	pTree, err := headCommit.Tree()
	if err != nil {
		return nil, err
	}

	commits := gitobject.NewCommitPostorderIter(headCommit, nil)

	err = commits.ForEach(func(commit *gitobject.Commit) error {
		log.Print(commit)

		cTime := commit.Author.When
		cTree, err := commit.Tree()
		if err != nil {
			return err
		}

		changes, err := cTree.Diff(pTree)
		if err != nil {
			return err
		}

		log.Printf("%s", len(changes))
		for _, change := range changes {
			if updateTimes.Update(change.To.Name, pTime) {
				log.Printf("Set last update time of %s to %s", change.To.Name, pTime)
			}

			if updateTimes.Update(change.From.Name, cTime) {
				log.Printf("Set last update time of %s to %s", change.From.Name, cTime)
			}

		}
		pTree = cTree
		pTime = cTime

		return nil
	})

	return updateTimes, err
}

func (g *GitStore) LoadObject(f *gitobject.File, updateTime time.Time) error {
	blobreader, err := f.Blob.Reader()
	contents, err := ioutil.ReadAll(blobreader)
	if err != nil {
		return err
	}

	log.Printf("Git path: %s", f.Name)
	t := strings.Split(f.Name, "/")[0]
	log.Printf("Looking up object type for: %s", t)
	switch t {
	case "node":
		obj := types.NewNode()
		err = yaml.Unmarshal(contents, &obj)
		if err != nil {
			return err
		}
		obj.LastUpdated = updateTime
		log.Printf("%s", obj)
		return g.cache.Update(obj)
	case "system":
		obj := types.NewSystem()
		err = yaml.Unmarshal(contents, &obj)
		if err != nil {
			return err
		}
		obj.LastUpdated = updateTime
		log.Printf("%s", obj)
		return g.cache.Update(obj)
	case "network":
		obj := types.NewNetwork()
		err = yaml.Unmarshal(contents, &obj)
		if err != nil {
			return err
		}
		obj.LastUpdated = updateTime
		log.Printf("%s", obj)
		return g.cache.Update(obj)
	default:
		log.Printf("No matching object type found")
		return nil
	}

}

func (g *GitStore) GetNodes() (map[string]*types.Node, error) {
	return g.cache.nodes, nil
}

func (g *GitStore) GetNetworks() (map[string]*types.Network, error) {
	return g.cache.networks, nil
}

func (g *GitStore) GetSystems() (map[string]*types.System, error) {
	return g.cache.systems, nil
}

func (g *GitStore) Update(obj interface{}) error {
	return fmt.Errorf("Not implemented")
}

func (g *GitStore) Delete(obj interface{}) error {
	return fmt.Errorf("Not implemented")
}

func (g *GitStore) Refresh() error {
	return g.Fetch(g.fetchOptions)
}
