// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package mocks to create the mocks run go generate to regenerate this package.
package mocks

//go:generate /usr/bin/env bash -c "echo \"$(cat ../../../hack/boilerplate/boilerplate.generatego.txt)\n$(../../../bin/mockgen -package=mocks github.com/mattermost/mattermost-cloud/internal/supervisor AWATClient)\" > client.go"
