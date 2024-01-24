package git

import (
	log "github.com/sirupsen/logrus"
)

// noopClient is a Git Client that is not configured.
type noopClient struct{}

// func (gt *noopClient) Checkout(branchName string, logger log.FieldLogger) error {
// 	logger.Debug("Git client is not configured; skipping checkout repository")

// 	return nil
// }

func (gt *noopClient) Commit(filePath, commitMsg string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Commit to the repository")

	return nil
}

func (gt *noopClient) Push(branchName string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Push to the repository")

	return nil
}

func (gt *noopClient) Close(tempDir string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Close the repository")

	return nil
}

func (gt *noopClient) Pull(logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Pull the repository")
	return nil
}
