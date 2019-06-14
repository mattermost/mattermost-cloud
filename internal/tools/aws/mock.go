package aws

type mockAPI struct {
	returnedError     error
	returnedTruncated bool
}
