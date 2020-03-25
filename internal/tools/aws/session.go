package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

// NewAWSSessionWithLogger initializes an AWS session instance with logging handler for debuging only.
func NewAWSSessionWithLogger(config *aws.Config, logger log.FieldLogger) (*session.Session, error) {
	awsSession, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	awsSession.Handlers.Complete.PushFront(func(r *request.Request) {
		loggingMessage := fmt.Sprintf("%s.%s %s %s: %s%s  %q", r.ClientInfo.ServiceID, r.Operation.Name, r.HTTPRequest.Method,
			r.HTTPResponse.Status, r.HTTPRequest.URL.Host, r.HTTPRequest.URL.RawPath, marshallParams(r.Params))

		if r.HTTPResponse.StatusCode >= 400 {
			logger.Error(loggingMessage)
		}

		if r.HTTPResponse.StatusCode < 400 {
			logger.Debug(loggingMessage)
		}
	})

	return awsSession, nil
}

func marshallParams(v interface{}) string {
	paramBytes, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(paramBytes)
}
