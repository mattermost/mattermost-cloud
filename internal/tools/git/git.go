package git

import (
	"os"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
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
	branchName string
}

// func (g *Git) Checkout(branchName string, logger log.FieldLogger) error {
// 	err := g.repo.Fetch(&git.FetchOptions{
// 		RefSpecs: []config.RefSpec{
// 			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/remotes/origin/%s", branchName, branchName)),
// 		},
// 		Auth: g.auth,
// 	})
// 	if err != nil && err != git.NoErrAlreadyUpToDate {
// 		return errors.Wrapf(err, "unable to fetch branch %s", branchName)
// 	}

// 	w, err := g.repo.Worktree()
// 	if err != nil {
// 		return errors.Wrap(err, "unable to create worktree")
// 	}

// 	err = w.Checkout(&git.CheckoutOptions{
// 		Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)),
// 		Create: true,
// 		Force:  true,
// 	})
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to checkout repository to branch: %v", branchName)
// 	}
// 	logger.Debugf("Checkout branch %s successfully", branchName)

// 	return nil
// }

func (g *Git) Pull(logger log.FieldLogger) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "unable to create worktree")
	}

	err = w.Pull(&git.PullOptions{
		Auth:          g.auth,
		RemoteName:    "origin",
		ReferenceName: plumbing.ReferenceName("refs/heads/" + g.branchName),
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return errors.Wrap(err, "unable to pull changes from the repository")
	}

	return nil
}

func (g *Git) Commit(filePath, commitMsg string, logger log.FieldLogger) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "unable to create worktree")
	}

	err = w.AddWithOptions(&git.AddOptions{
		All:  true,
		Path: filePath,
	})
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

func (g *Git) Push(branchName string, logger log.FieldLogger) error {
	remote, err := g.repo.Remote("origin")
	if err != nil {
		return errors.Wrapf(err, "unable to get remote origin")
	}

	err = remote.Push(&git.PushOptions{
		Auth:     g.auth,
		Progress: os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec("refs/heads/" + branchName + ":refs/heads/" + branchName),
		},
	})

	if err != nil {
		//return errors.Wrapf(err, "unable to push changes to the repository")
		logger.Warnf("Unable to push changes to the repository: %v", err)
	}
	logger.Debug("Push to repository successfully")

	return nil
}

func (g *Git) Close(tempDir string, logger log.FieldLogger) error {
	logger.Debugf("Remove temporary git directory: %s", tempDir)
	return os.RemoveAll(tempDir)
}
