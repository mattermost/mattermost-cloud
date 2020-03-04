package aws

// WARNING:
// This test is meant to exercise the provisioning and teardown of an AWS S3
// filestore in a real AWS account. Only set the test env vars below if you wish
// to test this process with real AWS resources.

// func TestFilestoreProvision(t *testing.T) {
// 	id := os.Getenv("SUPER_AWS_FILESTORE_TEST")
// 	if id == "" {
// 		return
// 	}

// 	logger := logrus.New()
// 	filestore := NewS3Filestore(id)

// 	logger.Warnf("Provisioning down AWS filestore %s", id)

// 	err := filestore.Provision(logger)
// 	require.NoError(t, err)
// }

// func TestFilestoreTeardown(t *testing.T) {
// 	id := os.Getenv("SUPER_AWS_FILESTORE_TEST")
// 	if id == "" {
// 		return
// 	}

// 	logger := logrus.New()
// 	filestore := NewS3Filestore(id)

// 	logger.Warnf("Tearing down AWS filestore %s", id)

// 	err := filestore.Teardown(false, logger)
// 	require.NoError(t, err)
// }
