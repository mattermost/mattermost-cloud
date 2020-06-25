// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package terraform

// backendFile contains the contents of the terraform file used to configure
// terraform remote state.
const backendFile = `
terraform {
    backend "s3" {}
}
`
