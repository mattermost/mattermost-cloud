// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

import (
	acmTypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Tag struct {
	Key   string
	Value string
}

func (t *Tag) toACM() acmTypes.Tag {
	return acmTypes.Tag{
		Key:   &t.Key,
		Value: &t.Value,
	}
}

func (t *Tag) toEC2() ec2Types.Tag {
	return ec2Types.Tag{
		Key:   &t.Key,
		Value: &t.Value,
	}
}

func NewTag(key, value string) Tag {
	return Tag{
		Key:   key,
		Value: value,
	}
}
