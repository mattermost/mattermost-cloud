// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

func parseEnvVarInput(rawInput []string, clear bool) (model.EnvVarMap, error) {
	if len(rawInput) != 0 && clear {
		return nil, errors.New("both mattermost-env and mattermost-env-clear were set; use one or the other")
	}
	if clear {
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

func registerPagingFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page", 0, "The page to fetch, starting at 0.")
	cmd.Flags().Int("per-page", 100, "The number of objects to fetch per page.")
	cmd.Flags().Bool("include-deleted", false, "Whether to include deleted objects.")
}

func getPagingModel(pf pagingFlags) model.Paging {
	return model.Paging{
		Page:           pf.page,
		PerPage:        pf.perPage,
		IncludeDeleted: pf.includeDeleted,
	}
}

func parsePagingFlags(cmd *cobra.Command) model.Paging {
	page, _ := cmd.Flags().GetInt("page")
	perPage, _ := cmd.Flags().GetInt("per-page")
	includeDeleted, _ := cmd.Flags().GetBool("include-deleted")

	return model.Paging{
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
	}
}
