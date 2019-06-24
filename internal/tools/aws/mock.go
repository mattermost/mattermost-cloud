package aws

// apiInterface abstracts out AWS API calls for testing.
type apiInterface struct{}

type mockAPI struct {
	returnedError     error
	returnedTruncated bool
}
