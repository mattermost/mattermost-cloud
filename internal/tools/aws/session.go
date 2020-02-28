package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

var awsSession *session.Session

// NewAWSSession creates a new AWS session if no error occurs.
func NewAWSSession() (*session.Session, error) {
	if awsSession != nil {
		return awsSession, nil
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// MargeSessionConfig add the supplied config to an existent config.
func MargeSessionConfig(config *aws.Config) {
	awsSession.Config.MergeIn(config)
}

// AddSessionLoggingHandler creates a handler for loggin calls to AWS.
func AddSessionLoggingHandler(logger log.FieldLogger) {
	awsSession.Handlers.Send.PushFront(func(r *request.Request) {
		logger.Debugf("%s: %s%s\n%s", r.HTTPRequest.Method, r.HTTPRequest.URL.Host, r.HTTPRequest.URL.RawPath, r.Params)
	})
}
