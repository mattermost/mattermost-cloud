package argocd

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
)

const (
	// DefaultTimeout is the default timeout for waiting for an application to be healthy.
	DefaultTimeout = 420 * time.Second
)

func (c *ApiClient) SyncApplication(gitopsAppName string) (*argoappv1.Application, error) {
	app, err := c.appClient.Sync(context.Background(), &application.ApplicationSyncRequest{
		Name: &gitopsAppName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to sync application.")
	}

	err = c.WaitForAppHealthy(gitopsAppName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wait for application to be healthy")
	}

	c.logger.Debugf("Successfully synced application %s", gitopsAppName)

	// This time is needed for the application to be available in the ArgoCD.
	c.logger.Debugf("Waiting for application %s to be available in the ArgoCD ...", gitopsAppName)
	time.Sleep(time.Second * 5)

	return app, nil
}

func (c *ApiClient) WaitForAppHealthy(appName string) error {

	c.logger.Infof("Waiting for application %s to be healthy ...", appName)

	startTime := time.Now()
	refresh := "true"

	for {
		app, err := c.appClient.Get(context.Background(), &application.ApplicationQuery{
			Name:    &appName,
			Refresh: &refresh,
		})

		if err == nil {
			if app.Status.Health.Status == health.HealthStatusHealthy && app.Status.Sync.Status == argoappv1.SyncStatusCodeSynced {
				break
			}
		}

		// Check for timeout
		if time.Since(startTime) >= DefaultTimeout {
			return errors.New("timed out waiting for application to be healthy")
		}

		//Add a small delay to reduce CPU usage and avoid too_many_pings error.
		//This time is needed for the application to be healthy in the ArgoCD.
		time.Sleep(time.Second * 5)
	}
	return nil
}
