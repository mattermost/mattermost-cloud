// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

func parseEnvVarInput(rawInput []string, clearEnv bool) (model.EnvVarMap, error) {
	if len(rawInput) != 0 && clearEnv {
		return nil, errors.New("both mattermost-env and mattermost-env-clear were set; use one or the other")
	}
	if clearEnv {
		// An empty non-nil map is what the API expects for a full env wipe.
		return make(model.EnvVarMap), nil
	}
	if len(rawInput) == 0 {
		return nil, nil
	}

	envVarMap := make(model.EnvVarMap)

	for _, env := range rawInput {
		// Split the input once by "=" to allow for multiple "="s to be in the
		// value. Expect there to still be one key and value.
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 || len(kv[0]) == 0 {
			return nil, errors.Errorf("%s is not in a valid env format; expecting KEY_NAME=VALUE", env)
		}

		if _, ok := envVarMap[kv[0]]; ok {
			return nil, errors.Errorf("env var %s was defined more than once", kv[0])
		}

		envVarMap[kv[0]] = model.EnvVar{Value: kv[1]}
	}

	return envVarMap, nil
}

func getPaging(pf pagingFlags) model.Paging {
	return model.Paging{
		Page:           pf.page,
		PerPage:        pf.perPage,
		IncludeDeleted: pf.includeDeleted,
	}
}

func runDryRun(request interface{}) error {
	if err := printJSON(request); err != nil {
		return errors.Wrap(err, "failed to print API request")
	}
	return nil
}

func createClient(flags clusterFlags) *model.Client {
	if len(flags.headers) > 0 {
		headers := make(map[string]string, len(flags.headers))
		for key, value := range flags.headers {
			headers[key] = value
		}
		return model.NewClientWithHeaders(flags.serverAddress, headers)
	}

	return model.NewClient(flags.serverAddress)
}
