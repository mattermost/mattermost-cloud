package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

// apiInterface abstracts out AWS API calls for testing.
type apiInterface struct{}

type mockAPI struct {
	returnedDescribeImagesOutput *ec2.DescribeImagesOutput
	returnedError                error
	returnedTruncated            bool
}
