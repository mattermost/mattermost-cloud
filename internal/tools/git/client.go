package git

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Client interface {
	Commit(filePath, commitMsg string, logger log.FieldLogger) error
	Push(branchName string, logger log.FieldLogger) error
	Close(tempDir string, logger log.FieldLogger) error
	Pull(logger log.FieldLogger) error
}

// Git is a wrapper around go-git to provide a simple interface for interacting with git.
// TODO: Handle concurrent access to the repo.
func NewGitClient(oauthToken, tempDir, remoteURL, authorName, branchName string) (Client, error) {
	if len(oauthToken) == 0 {
		return &NoOpClient{}, errors.New("no git token provided")
	}
	auth := &http.BasicAuth{
		Username: "provisioner",
		Password: oauthToken,
	}

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:           remoteURL,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branchName),
		Auth:          auth,
	})

	if err != nil {
		return nil, errors.Wrapf(err, "unable to clone repository:, %s", remoteURL)
	}

	return &Git{
		auth:       auth,
		repo:       repo,
		authorName: authorName,
		branchName: branchName,
	}, nil
}
