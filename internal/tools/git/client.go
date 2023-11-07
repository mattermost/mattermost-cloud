package git

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Client interface {
	Checkout(branchName string, logger log.FieldLogger) error
	Commit(filePath, commitMsg, authorName string, logger log.FieldLogger) error
	Push(logger log.FieldLogger) error
	Close(tempDir string, logger log.FieldLogger) error
}

func NewGitClient(oauthToken, tempDir, remoteURL string) (Client, error) {
	if len(oauthToken) == 0 {
		return &noopClient{}, errors.New("no git token provided")
	}
	auth := &http.BasicAuth{
		Username: "provisioner",
		Password: oauthToken,
	}

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:  remoteURL,
		Auth: auth,
	})

	if err != nil {
		return nil, errors.Wrapf(err, "unable to clone repository:, %s", remoteURL)
	}

	return &Git{
		auth: auth,
		repo: repo,
	}, nil
}
