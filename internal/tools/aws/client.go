package aws

import "github.com/aws/aws-sdk-go/service/route53"

// Client is a client for interacting with AWS resources.
type Client struct {
	hostedZoneID string
	api          api
}

// api mocks out the AWS API calls for testing.
type api interface {
	getSessionClient() (*route53.Route53, error)
	changeResourceRecordSets(*route53.Route53, *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
	listResourceRecordSets(*route53.Route53, *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error)
}

// New returns a new AWS client.
func New(hostedZoneID string) *Client {
	return &Client{
		hostedZoneID: hostedZoneID,
		api:          &apiInterface{},
	}
}
