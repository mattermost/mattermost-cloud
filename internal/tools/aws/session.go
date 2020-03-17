package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

// NewAWSSessionWithLogger initialized an AWS session instance with logging handler for debuging only.
func NewAWSSessionWithLogger(config *aws.Config, logger log.FieldLogger) (*session.Session, error) {
	awsSession, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	awsSession.Handlers.Send.PushFront(func(r *request.Request) {
		logger.Debugf("%s: %s%s\n%s", r.HTTPRequest.Method, r.HTTPRequest.URL.Host, r.HTTPRequest.URL.RawPath, r.Params)
	})

	return awsSession, nil
}
