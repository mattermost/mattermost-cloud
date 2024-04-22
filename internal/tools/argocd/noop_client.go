package argocd

import (
	"sync"
	"time"

	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type NoOpClient struct{}

func (n *NoOpClient) SyncApplication(gitopsAppName string) (*argoappv1.Application, error) {
	return nil, nil
}

func (n *NoOpClient) WaitForAppHealthy(appName string, wg *sync.WaitGroup, timeout time.Duration) error {
	return nil
}
