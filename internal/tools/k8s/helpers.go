package k8s

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
)

// WaitForPodRunning will poll a given kubernetes pod at a regular interval for
// it to enter the 'Running' state. If the pod fails to become ready before
// the provided timeout then an error will be returned.
func (kc *KubeClient) WaitForPodRunning(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	for {
		pod, err := kc.Clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err == nil {
			if pod.Status.Phase == corev1.PodRunning {
				return pod, nil
			}
		}

		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "timed out waiting for pod to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}

// GetPodsFromDeployment gets the pods that belong to a given deployment.
func (kc *KubeClient) GetPodsFromDeployment(namespace, deploymentName string) (*corev1.PodList, error) {
	deployment, err := kc.Clientset.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	set := labels.Set(deployment.GetLabels())
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}

	return kc.Clientset.CoreV1().Pods(namespace).List(listOptions)
}

// RemoteCommand executes a kubernetes command against a remote cluster.
func (kc *KubeClient) RemoteCommand(method string, url *url.URL) ([]byte, error) {
	exec, err := remotecommand.NewSPDYExecutor(kc.GetConfig(), method, url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute remote command")
	}

	var stdin io.Reader
	var stdout, stderr bytes.Buffer

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
			return nil, errors.Errorf("remote command failed with exit status %d: %s%s", exitErr.ExitStatus(), stdout.String(), stderr.String())
		}

		return nil, errors.Wrapf(err, "remote command failed: %s%s", stdout.String(), stderr.String())
	}

	return stdout.Bytes(), nil
}
