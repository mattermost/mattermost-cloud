package helm

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/pkg/errors"
)

func addLoadBalancerNameTag(elbClient aws.ELB, hostname string) error {
	if hostname == "" {
		return errors.New("cannot add loadbalancer name tag if hostname is empty")
	}

	parts := strings.Split(hostname, "-")
	loadbalancerName := parts[0]

	resource, err := elbClient.GetLoadBalancerResource(loadbalancerName)
	if err != nil {
		return errors.Wrap(err, "failed to get loadbalancer ARN")
	}

	err = elbClient.TagLoadBalancer(resource, map[string]string{
		"Name": loadbalancerName,
	})
	if err != nil {
		return errors.Wrap(err, "failed to tag loadbalancer")
	}

	return nil
}
