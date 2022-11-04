package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
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
	assert.Equal(t, []*rds.Tag{
		{
			Key:   &key,
			Value: &value,
		},
		{
			Key:   &key2,
			Value: &value2,
		},
	}, tags.ToRDSTags())
}

func TestTagsASEC2Tags(t *testing.T) {
	key := "key"
	value := "value"
	key2 := "key2"
	value2 := "value2"
	tags, err := NewTags(key, value, key2, value2)
	assert.NoError(t, err)
	assert.Equal(t, []*ec2.Tag{
		{
			Key:   &key,
			Value: &value,
		},
		{
			Key:   &key2,
			Value: &value2,
		},
	}, tags.ToEC2Tags())
}
