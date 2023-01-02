// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package mockawstools to create the mocks run go generate to regenerate this package.
//
//go:generate ../../../bin/mockgen -package=mockawstools -destination ./client.go -source ../../tools/aws/client.go
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt client.go > _client.go && mv _client.go client.go"
package mockawstools //nolint
