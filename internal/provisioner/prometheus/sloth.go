package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PrepareSloth prepares sloth resources after prometheus helm is installed.
func PrepareSloth(k8sClient *k8s.KubeClient, logger logrus.FieldLogger) error {
	files := []k8s.ManifestFile{
		{
			Path:            "manifests/sloth/crd_sloth.slok.dev_prometheusservicelevels.yaml",
			DeployNamespace: Namespace,
		},
		{
			Path:            "manifests/sloth/sloth.yaml",
			DeployNamespace: Namespace,
		},
	}

	err := k8sClient.CreateFromFiles(files)
	if err != nil {
		return errors.Wrapf(err, "failed to create sloth resources.")
	}
	wait := 240
	pods, err := k8sClient.GetPodsFromDeployment(Namespace, "sloth")
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found from sloth deployment")
	}

	for _, pod := range pods.Items {
		logger.Infof("Waiting up to %d seconds for %q pod %q to start...", wait, "sloth", pod.GetName())
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
		defer cancel()
		_, err := k8sClient.WaitForPodRunning(ctx, Namespace, pod.GetName())
		if err != nil {
			return err
		}
		logger.Infof("Successfully deployed service pod %q", pod.GetName())
	}

	return nil
}
