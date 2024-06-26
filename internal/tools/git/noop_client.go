package git

import (
	log "github.com/sirupsen/logrus"
)

// noopClient is a Git Client that is not configured.
type NoOpClient struct{}

func (n *NoOpClient) Commit(filePath, commitMsg string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Commit to the repository")

	return nil
}

func (n *NoOpClient) Push(logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Push to the repository")

	return nil
}

func (n *NoOpClient) Close(tempDir string, logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Close the repository")

	return nil
}

func (n *NoOpClient) Pull(logger log.FieldLogger) error {
	logger.Debug("Git client is not configured; skipping Pull the repository")
	return nil
}
