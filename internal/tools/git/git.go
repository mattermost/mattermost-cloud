package git

import (
	"os"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Git struct {
	repo       *git.Repository
	auth       http.AuthMethod
	authorName string
}

func (g *Git) Checkout(branchName string, logger log.FieldLogger) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "unable to create worktree")
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branchName),
		Force:  true,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to checkout repository to branch: %v", branchName)
	}
	logger.Debugf("Checkout branch %s successfully", branchName)

	return nil
}

func (g *Git) Commit(filePath, commitMsg string, logger log.FieldLogger) error {

	w, err := g.repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "unable to create worktree")
	}

	_, err = w.Add(filePath)
	if err != nil {
		return errors.Wrapf(err, "unable to add file to the worktree: %v", filePath)
	}

	commitSHA, err := w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name: g.authorName,
			When: time.Now(),
		},
	})
	if err != nil {
		return errors.Wrapf(err, "unable to commit changes to the repository %v:", w.Filesystem.Root())
	}
	logger.Debugf("Git commit successfully, sha: %s", commitSHA.String())

	return nil
}

func (g *Git) Push(logger log.FieldLogger) error {
	err := g.repo.Push(&git.PushOptions{
		Auth:     g.auth,
		Progress: os.Stdout,
	})

	if err != nil {
		return errors.Wrapf(err, "unable to push changes to the repository")
	}
	logger.Debug("Push to repository successfully")

	return nil
}

func (g *Git) Close(tempDir string, logger log.FieldLogger) error {
	logger.Debugf("Remove temporary git directory: %s", tempDir)
	return os.RemoveAll(tempDir)
}
