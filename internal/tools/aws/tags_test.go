// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"testing"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
)

func TestTagNewWithMany(t *testing.T) {
	tags, err := NewTags("one", "two")
	assert.NoError(t, err)
	assert.Len(t, tags.tags, 1)
	tags, err = NewTags("one", "two", "three", "four")
	assert.NoError(t, err)
	assert.Len(t, tags.tags, 2)
}

func TestTagLen(t *testing.T) {
	tags, err := NewTags("one", "two")
	assert.NoError(t, err)
	assert.Equal(t, 1, tags.Len())
	tags.Add("other", "tag")
	assert.Equal(t, 2, tags.Len())
}

func TestTagAdd(t *testing.T) {
	tags, err := NewTags()
	assert.NoError(t, err)
	tags.Add("key", "value")
	assert.Len(t, tags.tags, 1)
}

func TestTagAddMany(t *testing.T) {
	tags, err := NewTags()
	assert.NoError(t, err)
	err = tags.AddMany("key", "value", "key2", "value2")
	assert.NoError(t, err)
	assert.Len(t, tags.tags, 2)
}

func TestTagAddManyOdd(t *testing.T) {
	tags, err := NewTags()
	assert.NoError(t, err)
	err = tags.AddMany("key", "value", "key2")
	assert.Error(t, err)
}

func TestTagsASRDSTags(t *testing.T) {
	key := "key"
	value := "value"
	key2 := "key2"
	value2 := "value2"
	tags, err := NewTags(key, value, key2, value2)
	assert.NoError(t, err)
	result := tags.ToRDSTags()
	assert.Subset(t, []rdsTypes.Tag{
		{
			Key:   &key,
			Value: &value,
		},
		{
			Key:   &key2,
			Value: &value2,
		},
	}, result)
	assert.Len(t, result, 2)
}

func TestTagsASEC2Tags(t *testing.T) {
	key := "key"
	value := "value"
	key2 := "key2"
	value2 := "value2"
	tags, err := NewTags(key, value, key2, value2)
	assert.NoError(t, err)
	result := tags.ToEC2Tags()
	assert.Subset(t, []ec2Types.Tag{
		{
			Key:   &key,
			Value: &value,
		},
		{
			Key:   &key2,
			Value: &value2,
		},
	}, result)
	assert.Len(t, result, 2)
}
