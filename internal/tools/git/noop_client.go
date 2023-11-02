package git

import (
	log "github.com/sirupsen/logrus"
)

// noopClient is a Git Client that is not configured.
type noopClient struct{}

func (gt *noopClient) Checkout(branchName string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping checkout repo")

	return nil
}

func (gt *noopClient) Commit(filePath, commitMsg, authorName string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping checkout repo")

	return nil
}

func (gt *noopClient) Push(logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping checkout repo")

	return nil
}

func (gt *noopClient) Close(tempDir string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping checkout repo")

	return nil
}
