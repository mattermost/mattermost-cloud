package aws

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var id = "test-id-6"

func TestDatabaseProvision(t *testing.T) {
	if os.Getenv("SUPER_TEST") == "" {
		return
	}

	logger := logrus.New()

	database := NewRDSDatabase(id)

	err := database.Provision(logger)
	require.NoError(t, err)
}

func TestDatabaseTeardown(t *testing.T) {
	if os.Getenv("SUPER_TEST") == "" {
		return
	}

	logger := logrus.New()

	database := NewRDSDatabase(id)

	err := database.Teardown(false, logger)
	require.NoError(t, err)
}
