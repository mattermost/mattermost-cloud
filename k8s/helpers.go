// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
)

// PatchStringValue is a helper struct for patch operations
type PatchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// WaitForPodRunning will poll a given kubernetes pod at a regular interval for
// it to enter the 'Running' state. If the pod fails to become ready before
// the provided timeout then an error will be returned.
func (kc *KubeClient) WaitForPodRunning(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	for {
		pod, err := kc.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err == nil {
			if pod.Status.Phase == corev1.PodRunning {
				return pod, nil
			}
		}
		if err != nil && k8serrors.IsNotFound(err) {
			kc.logger.Infof("Pod %s not found in %s namesapace, maybe was part of the old replicaset, since we updated the deployment/statefullsets, moving on", podName, namespace)
			return &corev1.Pod{}, nil
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
	ctx := context.TODO()
	deployment, err := kc.Clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	set := labels.Set(deployment.GetLabels())
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}

	return kc.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
}

// GetPodsFromStatefulset gets the pods that belong to a given stateful set.
func (kc *KubeClient) GetPodsFromStatefulset(namespace, statefulSetName string) (*corev1.PodList, error) {
	ctx := context.TODO()
	statefulSet, err := kc.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	set := labels.Set(statefulSet.GetLabels())
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}

	return kc.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
}

// GetPodsFromDaemonSet gets the pods that belong to a given daemonset.
func (kc *KubeClient) GetPodsFromDaemonSet(namespace, daemonSetName string) (*corev1.PodList, error) {
	ctx := context.TODO()
	daemonSet, err := kc.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, daemonSetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	set := labels.Set(daemonSet.GetLabels())
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}

	return kc.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
}

// PatchPodsDaemonSet patches the pods that belong to a given daemonset with a given payload.
func (kc *KubeClient) PatchPodsDaemonSet(namespace, daemonSetName string, payload []PatchStringValue) error {
	ctx := context.TODO()
	daemonSet := kc.Clientset.AppsV1().DaemonSets(namespace)
	payloadBytes, _ := json.Marshal(payload)
	_, err := daemonSet.Patch(ctx, daemonSetName, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		errors.Wrapf(err, "failed to patch daemonSet %s", daemonSetName)
		return err
	}
	return nil
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
	output := append(stdout.Bytes(), stderr.Bytes()...)

	if err != nil {
		if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
			return output, errors.Errorf("remote command failed with exit status %d: %s%s", exitErr.ExitStatus(), stdout.String(), stderr.String())
		}

		return output, errors.Wrapf(err, "remote command failed: %s%s", stdout.String(), stderr.String())
	}

	return output, nil
}
