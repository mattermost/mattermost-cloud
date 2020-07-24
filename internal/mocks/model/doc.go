// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:generate mockgen -package=mocks -destination ./installation_database.go -source ../../../model/installation_database.go
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt installation_database.go > _installation_database.go && mv _installation_database.go installation_database.go"
package mocks
