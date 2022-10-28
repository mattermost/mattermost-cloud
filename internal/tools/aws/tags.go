package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
)

// Tags an abstract represtation of tags that can be converted to different AWS resource tags
type Tags struct {
	tags map[string]string
}

// Add adds a new tag in a key,value format
func (t *Tags) Add(key, value string) {
	t.tags[key] = value
}

// AddMany adds an indetermited amount of tags, must be even
func (t *Tags) AddMany(items ...string) error {
	if len(items)%2 != 0 {
		return errors.New("add many requires an even number of arguments")
	}

	for i := 0; i < len(items); i = i + 2 {
		t.tags[items[i]] = items[i+1]
	}

	return nil
}

// ToRDSTags convert the tags into an RDS tags format
func (t *Tags) ToRDSTags() []*rds.Tag {
	result := make([]*rds.Tag, 0, t.Len())

	for k, v := range t.tags {
		result = append(result, &rds.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	return result
}

// ToEC2Tags convert the tags into an RDS tags format
func (t *Tags) ToEC2Tags() []*ec2.Tag {
	result := make([]*ec2.Tag, 0, t.Len())

	for k, v := range t.tags {
		result = append(result, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	return result
}

// Len returns the number of tags
func (t *Tags) Len() int {
	return len(t.tags)
}

// NewTags create a new instance of AWSTags optionally adding some of them on creation
func NewTags(items ...string) (*Tags, error) {
	t := Tags{
		tags: make(map[string]string),
	}

	if err := t.AddMany(items...); err != nil {
		return nil, err
	}

	return &t, nil
}
