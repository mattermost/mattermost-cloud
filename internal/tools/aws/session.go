package aws

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

var awsSession *session.Session
var lock = &sync.Mutex{}
var err error

// NewAWSSession creates a new AWS session if no error occurs.
func NewAWSSession() (*session.Session, error) {
	if awsSession == nil {
		lock.Lock()
		defer lock.Unlock()
		awsSession, err = session.NewSession(&aws.Config{
			Region: aws.String(DefaultAWSRegion),

			// TODO(gsagula): we should use Retryer for a more robust retry strategy.
			// https://github.com/aws/aws-sdk-go/blob/99cd35c8c7d369ba8c32c46ed306f6c88d24cfd7/aws/request/retryer.go#L20
			MaxRetries: aws.Int(DefaultAWSClientRetries),
		})
	}

	if err != nil {
		return nil, err
	}

	return awsSession, nil
}

// NewAWSSessionWithLogger initialized a singleton AWS session instance with logging handler. This method should be called first in the code.
func NewAWSSessionWithLogger(logger log.FieldLogger) (*session.Session, error) {
	awsSession, err = NewAWSSession()

	awsSession.Handlers.Send.PushFront(func(r *request.Request) {
		logger.Debugf("%s: %s%s\n%s", r.HTTPRequest.Method, r.HTTPRequest.URL.Host, r.HTTPRequest.URL.RawPath, r.Params)
	})

	return awsSession, nil
}
