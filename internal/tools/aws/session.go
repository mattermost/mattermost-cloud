package aws

import (
	"bytes"
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
		if r.HTTPResponse != nil && r.HTTPRequest != nil {
			var buffer bytes.Buffer
			buffer.WriteString(fmt.Sprintf("%s.%s %s %s: %s%s  ", r.ClientInfo.ServiceID, r.Operation.Name, r.HTTPRequest.Method,
				r.HTTPResponse.Status, r.HTTPRequest.URL.Host, r.HTTPRequest.URL.RawPath))

			paramBytes, err := json.Marshal(r.Params)
			if err != nil {
				buffer.WriteString(err.Error())
			} else {
				buffer.Write(paramBytes)
			}

			if r.HTTPResponse.StatusCode >= 400 {
				logger.Error(buffer.String())
			}

			if r.HTTPResponse.StatusCode < 400 {
				logger.Debug(buffer.String())
			}
		}
	})

	return awsSession, nil
}
