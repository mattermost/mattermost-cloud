// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"encoding/json"
	"strings"

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

// This is far from an ideal way to handle the sanitization
// but we can retain some AWS logs and debuggability without investing
// to much time into sophisticated sanitization/logging mechanism.
func sanitizeParams(reqParams interface{}, logger log.FieldLogger) string {
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
