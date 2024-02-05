package argocd

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
)

func (c *ApiClient) SyncApplication(gitopsAppName string) (*argoappv1.Application, error) {
	app, err := c.appClient.Sync(context.Background(), &application.ApplicationSyncRequest{
		Name: &gitopsAppName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to sync application.")
	}

	var wg sync.WaitGroup
	timeout := time.Second * 300
	wg.Add(1)
	go c.waitForSyncCompletion(gitopsAppName, &wg, timeout)
	wg.Wait()

	c.logger.Debugf("Successfully synced application %s", gitopsAppName)

	// This time is needed for the application to be available in the ArgoCD.
	// time.Sleep(time.Second * 10)

	return app, nil
}

func (c *ApiClient) WaitForAppHealthy(appName string, wg *sync.WaitGroup, timeout time.Duration) error { //TODO return error
	defer wg.Done()

	c.logger.Infof("Waiting for application %s to be synced...", appName)

	startTime := time.Now()
	refresh := "true"

	for {
		app, err := c.appClient.Get(context.Background(), &application.ApplicationQuery{
			Name:    &appName,
			Refresh: &refresh,
		})
		if err != nil {
			c.logger.Errorf("failed to get application %s: %v", appName, err)
			return errors.Wrap(err, "failed to get application.")
		}

		if app.Status.Health.Status == health.HealthStatusHealthy {
			break
		}

		// Check for timeout
		if time.Since(startTime) >= timeout {
			c.logger.Errorf("timed out waiting for application %s to be healthy", appName)
			return errors.New("timed out waiting for application to be healthy")
		}

		time.Sleep(time.Millisecond * 100)
	}
	return nil
}

func (c *ApiClient) waitForSyncCompletion(appName string, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()

	startTime := time.Now()
	refresh := "true"

	for {

		syncStatus, err := c.appClient.Get(context.Background(), &application.ApplicationQuery{
			Name:    &appName,
			Refresh: &refresh,
		})
		if err != nil {
			c.logger.Errorf("failed to get application %s: %v", appName, err)
		}

		if syncStatus.Status.OperationState.Phase != "Running" {
			break
		}

		c.logger.Infof("Waiting for application %s to be synced... STATUS: %s\n", appName, syncStatus.Status.OperationState.Phase)

		// Check for timeout
		if time.Since(startTime) >= timeout {
			c.logger.Errorf("timed out waiting for application %s to be healthy", appName)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}
