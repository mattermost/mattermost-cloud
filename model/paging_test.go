// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaging_AddToQuery(t *testing.T) {
	req := Paging{
		Page:           1,
		PerPage:        5,
		IncludeDeleted: true,
	}

	u, err := url.Parse("https://provisioner/api")
	require.NoError(t, err)
	q := u.Query()

	req.AddToQuery(q)

	assert.Equal(t, "1", q.Get("page"))
	assert.Equal(t, "5", q.Get("per_page"))
	assert.Equal(t, "true", q.Get("include_deleted"))
}
