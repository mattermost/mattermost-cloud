package aws

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
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

			buffer.WriteString(fmt.Sprintf("[aws] %s %s %s ", r.HTTPRequest.Method, r.HTTPResponse.Status, r.HTTPResponse.Request.URL.String()))

			paramBytes, err := json.Marshal(r.Params)
			if err != nil {
				buffer.WriteString(err.Error())
			} else {
				buffer.Write(paramBytes)
			}

			logger = logger.WithFields(logrus.Fields{
				"aws-service-id":     r.ClientInfo.ServiceID,
				"aws-operation-name": r.Operation.Name,
			})

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
