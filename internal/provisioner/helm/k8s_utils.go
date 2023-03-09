package helm

import (
	"context"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getElasticLoadBalancerInfo returns the private load balancer endpoint and type of the NGINX service.
func getElasticLoadBalancerInfo(namespace string, logger log.FieldLogger, configPath string) (string, string, error) {
	k8sClient, err := k8s.NewFromFile(configPath, logger)
	if err != nil {
		return "", "", err
	}

	services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", "", err
	}

	for _, service := range services.Items {
		if service.Spec.Type == "LoadBalancer" {
			if service.Status.LoadBalancer.Ingress != nil {
				endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
				if endpoint != "" {
					return endpoint, service.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"], nil
				}
			}
		}
	}

	return "", "", nil
}

// getPrivateLoadBalancerEndpoint returns the private load balancer endpoint of the NGINX service.
func getPrivateLoadBalancerEndpoint(ctx context.Context, namespace string, logger log.FieldLogger, configPath string) (string, error) {
	k8sClient, err := k8s.NewFromFile(configPath, logger)
	if err != nil {
		return "", err
	}

	for {
		services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return "", err
		}
		for _, service := range services.Items {
			if service.Spec.Type == "LoadBalancer" || strings.HasSuffix(service.Name, "query") {
				if service.Status.LoadBalancer.Ingress != nil {
					endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
					if endpoint == "" {
						return "", errors.New("loadbalancer endpoint value is empty")
					}

					return endpoint, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", errors.Wrap(ctx.Err(), "timed out waiting for internal load balancer to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}
