// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-operator/pkg/resources"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	ciExecJobTTLSeconds int32 = 180
)

// ExecMMCTL runs the given MMCTL command against the given cluster installation.
// Setup and exec errors both result in a single return error.
func (provisioner Provisioner) ExecMMCTL(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	output, execErr, err := provisioner.ExecClusterInstallationCLI(cluster, clusterInstallation, append([]string{"./bin/mmctl"}, args...)...)
	if err != nil {
		return output, errors.Wrap(err, "failed to run mmctl command")
	}
	if execErr != nil {
		return output, errors.Wrap(execErr, "mmctl command encountered an error")
	}

	return output, nil
}

// ExecClusterInstallationCLI execs the provided command on the defined cluster
// installation and returns both exec preparation errors as well as errors from
// the exec command itself.
func (provisioner Provisioner) ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get kube client")
	}

	return execClusterInstallationCLI(k8sClient, clusterInstallation, logger, args...)
}

// ExecMattermostCLI invokes the Mattermost CLI for the given cluster installation
// with the given args. Setup and exec errors both result in a single return error.
func (provisioner Provisioner) ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	output, execErr, err := provisioner.ExecClusterInstallationCLI(cluster, clusterInstallation, append([]string{"./bin/mattermost"}, args...)...)
	if err != nil {
		return output, errors.Wrap(err, "failed to run mattermost command")
	}
	if execErr != nil {
		return output, errors.Wrap(execErr, "mattermost command encountered an error")
	}

	return output, nil
}

// ExecClusterInstallationJob creates job executing command on cluster installation.
func (provisioner Provisioner) ExecClusterInstallationJob(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Executing job with CLI command on cluster installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kube client")
	}

	ctx := context.TODO()
	deploymentList, err := k8sClient.Clientset.AppsV1().Deployments(clusterInstallation.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return errors.Wrap(err, "failed to get installation deployments")
	}

	if len(deploymentList.Items) == 0 {
		return errors.New("no mattermost deployments found")
	}

	jobName := fmt.Sprintf("command-%s", uuid.New()[:6])
	job := resources.PrepareMattermostJobTemplate(jobName, clusterInstallation.Namespace, &deploymentList.Items[0], nil)
	// TODO: refactor above method in Mattermost Operator to take command and handle this logic inside.
	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Command = args
		// We want to match bifrost network policy so that server can come up quicker.
		job.Spec.Template.Labels["app"] = "mattermost"
	}
	jobTTL := ciExecJobTTLSeconds
	job.Spec.TTLSecondsAfterFinished = &jobTTL

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(clusterInstallation.Namespace)

	defer func() {
		errDefer := jobsClient.Delete(ctx, jobName, metav1.DeleteOptions{})
		if errDefer != nil && !k8sErrors.IsNotFound(errDefer) {
			logger.Errorf("Failed to cleanup exec job: %q", jobName)
		}
	}()

	job, err = jobsClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create CLI command job")
	}

	err = wait.Poll(time.Second, 10*time.Minute, func() (done bool, err error) {
		job, err = jobsClient.Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get %q job", jobName)
		}
		if job.Status.Succeeded < 1 {
			logger.Infof("job %q not yet finished, waiting up to 10 minute", jobName)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "job %q did not finish in expected time", jobName)
	}

	return nil
}

func execClusterInstallationCLI(k8sClient *k8s.KubeClient, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger, args ...string) ([]byte, error, error) {
	ctx := context.TODO()
	podList, err := k8sClient.Clientset.CoreV1().Pods(clusterInstallation.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=mattermost",
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query mattermost pods")
	}

	// In the future, we'd ideally just spin our own container on demand, allowing
	// configuration changes even if the pods are failing to start the server. For now,
	// we find the first pod running Mattermost, and pick the first container therein.

	if len(podList.Items) == 0 {
		return nil, nil, errors.New("failed to find mattermost pods on which to exec")
	}

	pod := podList.Items[0]
	if len(pod.Spec.Containers) == 0 {
		return nil, nil, errors.Errorf("failed to find containers in pod %s", pod.Name)
	}

	container := pod.Spec.Containers[0]
	logger.Debugf("Executing `%s` on pod %s: container=%s, image=%s, phase=%s", strings.Join(args, " "), pod.Name, container.Name, container.Image, pod.Status.Phase)

	now := time.Now()
	output, execErr := execCLI(k8sClient, clusterInstallation.Namespace, pod.Name, container.Name, args...)
	if execErr != nil {
		logger.WithError(execErr).Warnf("Command `%s` on pod %s finished in %.0f seconds, but encountered an error", strings.Join(args, " "), pod.Name, time.Since(now).Seconds())
	} else {
		logger.Debugf("Command `%s` on pod %s finished in %.0f seconds", strings.Join(args, " "), pod.Name, time.Since(now).Seconds())
	}

	return output, execErr, nil
}

func execCLI(k8sClient *k8s.KubeClient, namespace, podName, containerName string, args ...string) ([]byte, error) {

	execRequest := k8sClient.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   args,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	return k8sClient.RemoteCommand("POST", execRequest.URL())
}
