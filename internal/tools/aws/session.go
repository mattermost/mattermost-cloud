// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const redactedParam = "*****"

// Specifies AWS request parameters that should be sanitized from logs.
var sanitizeLogParams = map[string]bool{
	"secretstring":       true,
	"secretbinary":       true,
	"masterpassword":     true,
	"masteruserpassword": true,
	"masterusername":     true,
}

// NewAWSSessionWithLogger initializes an AWS session instance with logging handler for debuging only.
func NewAWSSessionWithLogger(config *aws.Config, logger log.FieldLogger) (*session.Session, error) {
	awsSession, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	awsSession.Handlers.Complete.PushFront(func(r *request.Request) {
		if r.HTTPResponse != nil && r.HTTPRequest != nil {
			var buffer bytes.Buffer

			buffer.WriteString(fmt.Sprintf("[aws] %s %s (%s)", r.HTTPRequest.Method, r.HTTPRequest.URL.String(), r.HTTPResponse.Status))

			if r.ParamsFilled() {
				params := sanitizeParams(r.Params, logger)
				logger = logger.WithField("params", params)
			}

			logger = logger.WithFields(logrus.Fields{
				"aws-service-id":     r.ClientInfo.ServiceID,
				"aws-operation-name": r.Operation.Name,
			})

			logger.Debug(buffer.String())
		}
	})

	return awsSession, nil
}

// This is far from an ideal way to handle the sanitization
// but we can retain some AWS logs and debuggability without investing
// to much time into sophisticated sanitization/logging mechanism.
func sanitizeParams(reqParams interface{}, logger logrus.FieldLogger) string {
	paramsJSON, err := json.Marshal(reqParams)
	if err != nil {
		logger.WithError(err).Warn("Failed to marshal AWS request parameters")
		return "<ALL_PARAMETERS_REDACTED>"
	}

	var paramMap map[string]interface{}
	err = json.Unmarshal(paramsJSON, &paramMap)
	if err != nil {
		logger.WithError(err).Warn("Failed to unmarshal AWS request parameters to map[string]interface{}")
		return "<ALL_PARAMETERS_REDACTED>"
	}

	for k := range paramMap {
		if sanitizeLogParams[strings.ToLower(k)] {
			paramMap[k] = redactedParam
		}
	}

	redactedJSON, err := json.Marshal(paramMap)
	if err != nil {
		logger.WithError(err).Warn("Failed to marshal sanitized AWS request params")
		return "<ALL_PARAMETERS_REDACTED>"
	}

	return string(redactedJSON)
}
