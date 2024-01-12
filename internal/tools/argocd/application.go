package argocd

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

func (c *AppClient) SyncApplication(appName string) (*argoappv1.Application, error) {
	var wg sync.WaitGroup
	timeout := time.Second * 300
	wg.Add(1)
	go c.waitForSyncCompletion(appName, &wg, timeout)

	wg.Wait()

	app, err := c.appClient.Sync(context.Background(), &application.ApplicationSyncRequest{
		Name: &appName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to sync application.")
	}

	c.logger.Debugf("Successfully synced application %s", appName)

	return app, nil
}

// func (c *AppClient) WaitForAppHealthy(appName string, wg *sync.WaitGroup, timeout time.Duration) {
// 	defer wg.Done()

// 	startTime := time.Now()

// 	for {
// 		app, err := c.appClient.Get(context.Background(), &application.ApplicationQuery{
// 			Name: &appName,
// 		})
// 		if err != nil {
// 			c.logger.Errorf("failed to get application %s: %v", appName, err)
// 			errors.Wrap(err, "failed to get application.")
// 		}

// 		if app.Status.Health.Status == health.HealthStatusHealthy {
// 			break
// 		}

// 		// Check for timeout
// 		if time.Since(startTime) >= timeout {
// 			c.logger.Errorf("timed out waiting for application %s to be healthy", appName)
// 			return
// 		}

// 		// Optional: Add a small delay to reduce CPU usage
// 		time.Sleep(time.Millisecond * 100)
// 	}
// }

func (c *AppClient) waitForSyncCompletion(appName string, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()

	startTime := time.Now()

	for {

		syncStatus, err := c.appClient.Get(context.Background(), &application.ApplicationQuery{
			Name: &appName,
		})
		if err != nil {
			c.logger.Errorf("failed to get application %s: %v", appName, err)
		}

		if syncStatus.Status.OperationState.Phase != "Running" {
			break
		}

		// Check for timeout
		if time.Since(startTime) >= timeout {
			c.logger.Errorf("timed out waiting for application %s to be healthy", appName)
			return
		}

		// Optional: Add a small delay to reduce CPU usage
		time.Sleep(time.Second * 30)
	}
}
