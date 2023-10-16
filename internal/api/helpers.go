// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/url"
	"strconv"

	"github.com/mattermost/mattermost-cloud/model"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func logSecurityLockConflict(resourceType string, logger logrus.FieldLogger) {
	logger.WithField("api-security-lock-conflict", resourceType).Warn("API security lock conflict detected")
}

func parseString(u *url.URL, name string, defaultValue string) string {
	valueStr := u.Query().Get(name)
	if valueStr == "" {
		return defaultValue
	}

	return valueStr
}

func parseInt(u *url.URL, name string, defaultValue int) (int, error) {
	valueStr := u.Query().Get(name)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s as integer", name)
	}

	return value, nil
}

func parseBool(u *url.URL, name string, defaultValue bool) (bool, error) {
	valueStr := u.Query().Get(name)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse %s as boolean", name)
	}

	return value, nil
}

func parsePaging(u *url.URL) (model.Paging, error) {
	page, err := parseInt(u, "page", 0)
	if err != nil {
		return model.Paging{}, err
	}

	perPage, err := parseInt(u, "per_page", 100)
	if err != nil {
		return model.Paging{}, err
	}

	includeDeleted, err := parseBool(u, "include_deleted", false)
	if err != nil {
		return model.Paging{}, err
	}

	return model.Paging{
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
	}, nil
}

func parseGroupConfig(u *url.URL) (bool, bool, error) {
	includeGroupConfig, err := parseBool(u, "include_group_config", true)
	if err != nil {
		return false, false, err
	}

	includeGroupConfigOverrides, err := parseBool(u, "include_group_config_overrides", true)
	if err != nil {
		return false, false, err
	}

	return includeGroupConfig, includeGroupConfigOverrides, nil
}

func parseDeletionLocked(u *url.URL) (*bool, error) {
	valueStr := u.Query().Get("deletion_locked")
	if valueStr == "" {
		return nil, nil
	}

	locked, err := strconv.ParseBool(valueStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deletion_locked boolean")
	}

	return &locked, nil
}
