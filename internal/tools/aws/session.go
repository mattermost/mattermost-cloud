package aws

const (
	defaultTTL    = 60
	defaultWeight = 1
)

// apiInterface abstracts out AWS API calls for testing.
type apiInterface struct{}
