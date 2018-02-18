package git

import (
	"os"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// ShallowClone does a git shallow clone.
func ShallowClone(path string, url string, ref string) (*git.Repository, error) {
	referenceName := "refs/heads/" + ref

	resp, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.ReferenceName(referenceName),
		Depth:         1,
		Progress:      os.Stdout,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
