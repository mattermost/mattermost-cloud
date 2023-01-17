// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestStandardMultitenantDatabaseTagFilters(t *testing.T) {
	databaseType := DefaultRDSMultitenantDatabasePerseusTypeTagValue
	engineType := DatabaseTypePostgresSQLAurora
	vpcID := model.NewID()

	tagFilters := standardMultitenantDatabaseTagFilters(databaseType, engineType, vpcID)
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(DefaultRDSMultitenantDatabaseTypeTagKey), databaseType, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(CloudInstallationDatabaseTagKey), engineType, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(VpcIDTagKey), vpcID, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(RDSMultitenantPurposeTagKey), RDSMultitenantPurposeTagValueProvisioning, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(RDSMultitenantOwnerTagKey), RDSMultitenantOwnerTagValueCloudTeam, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(DefaultAWSTerraformProvisionedKey), DefaultAWSTerraformProvisionedValueTrue, tagFilters))
	assert.False(t, ensureTagFilterInFilterSet(trimTagPrefix("key"), "value", tagFilters))

	databaseType = DefaultRDSMultitenantDatabaseDBProxyTypeTagValue
	vpcID = model.NewID()

	tagFilters = standardMultitenantDatabaseTagFilters(databaseType, engineType, vpcID)
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(DefaultRDSMultitenantDatabaseTypeTagKey), databaseType, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(CloudInstallationDatabaseTagKey), engineType, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(VpcIDTagKey), vpcID, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(RDSMultitenantPurposeTagKey), RDSMultitenantPurposeTagValueProvisioning, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(RDSMultitenantOwnerTagKey), RDSMultitenantOwnerTagValueCloudTeam, tagFilters))
	assert.True(t, ensureTagFilterInFilterSet(trimTagPrefix(DefaultAWSTerraformProvisionedKey), DefaultAWSTerraformProvisionedValueTrue, tagFilters))
	assert.False(t, ensureTagFilterInFilterSet(trimTagPrefix("key"), "value", tagFilters))
}
