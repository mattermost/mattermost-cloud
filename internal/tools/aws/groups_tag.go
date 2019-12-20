package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	groupstag "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
)

// TODO(gsagula): create interface for mocking and write tests.

// GroupsTagFilter holds key/values pairs to find tags in AWS.
type GroupsTagFilter []struct {
	Key    string
	Values []string
}

func (gtf *GroupsTagFilter) tagFilter() []*groupstag.TagFilter {
	tagFilter := make([]*groupstag.TagFilter, len(*gtf))
	for k, v := range *gtf {
		tagFilter[k] = &groupstag.TagFilter{
			Key:    aws.String(trimTagPrefix(v.Key)),
			Values: make([]*string, len(v.Values)),
		}
		for j, value := range v.Values {
			tagFilter[k].Values[j] = &value
		}
	}
	return tagFilter
}

// GetResources returns all resources associated to the filter.
func (gtf *GroupsTagFilter) GetResources() ([]*groupstag.ResourceTagMapping, error) {
	svc := groupstag.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	var resources []*groupstag.ResourceTagMapping
	var pagetoken *string
	for {
		out, err := svc.GetResources(&groupstag.GetResourcesInput{
			TagFilters:       gtf.tagFilter(),
			ResourcesPerPage: aws.Int64(100),
			PaginationToken:  pagetoken,
		})
		if err != nil {
			return nil, err
		}
		resources = append(resources, out.ResourceTagMappingList...)
		if out.PaginationToken == nil || *out.PaginationToken == "" {
			break
		}
		pagetoken = out.PaginationToken
	}

	return resources, nil
}
