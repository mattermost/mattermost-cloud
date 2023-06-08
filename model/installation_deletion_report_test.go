// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestDeletionPendingReport(t *testing.T) {
	now := time.Now()
	var report model.DeletionPendingReport
	report.NewCutoff("6 hours", now.Add(6*time.Hour))
	report.NewCutoff("12 hours", now.Add(12*time.Hour))

	t.Run("no counts", func(t *testing.T) {
		assert.Zero(t, report.Cutoffs[0].Count)
		assert.Zero(t, report.Cutoffs[1].Count)
		assert.Zero(t, report.Overflow)
	})

	t.Run("count 3 hours", func(t *testing.T) {
		report.Count(model.GetMillisAtTime(now.Add(3 * time.Hour)))
		assert.Equal(t, 1, report.Cutoffs[0].Count)
		assert.Zero(t, report.Cutoffs[1].Count)
		assert.Zero(t, report.Overflow)
	})

	t.Run("count 5 hours", func(t *testing.T) {
		report.Count(model.GetMillisAtTime(now.Add(5 * time.Hour)))
		assert.Equal(t, 2, report.Cutoffs[0].Count)
		assert.Zero(t, report.Cutoffs[1].Count)
		assert.Zero(t, report.Overflow)
	})

	t.Run("count 10 hours", func(t *testing.T) {
		report.Count(model.GetMillisAtTime(now.Add(10 * time.Hour)))
		assert.Equal(t, 2, report.Cutoffs[0].Count)
		assert.Equal(t, 1, report.Cutoffs[1].Count)
		assert.Zero(t, report.Overflow)
	})

	t.Run("count 24 hours", func(t *testing.T) {
		report.Count(model.GetMillisAtTime(now.Add(24 * time.Hour)))
		assert.Equal(t, 2, report.Cutoffs[0].Count)
		assert.Equal(t, 1, report.Cutoffs[1].Count)
		assert.Equal(t, 1, report.Overflow)
	})
}
