// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package mocks to create the mocks run go generate to regenerate this package.
//go:generate ../../../bin/mockgen -package=mocks -destination ./logrus.go github.com/sirupsen/logrus StdLogger,FieldLogger,Ext1FieldLogger
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt logrus.go > _logrus.go && mv _logrus.go logrus.go"
package mocks //nolint
