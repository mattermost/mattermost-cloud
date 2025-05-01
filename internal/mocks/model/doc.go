// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package mockmodel to create the mocks run go generate to regenerate this package.
//
//go:generate ../../../bin/mockgen -package=mockmodel -destination ./installation_database.go -source ../../../model/installation_database.go
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt installation_database.go > _installation_database.go && mv _installation_database.go installation_database.go"
package mockmodel
