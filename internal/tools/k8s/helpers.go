package k8s

import (
	"errors"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
)

// WaitForPodRunning will poll a given kubernetes pod at a regular interval for
// it to enter the 'Running' state. If the pod fails to become ready before
// the provided timeout then an error will be returned.
func (kc *KubeClient) WaitForPodRunning(name, namespace string, timeout int) (apiv1.Pod, error) {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return apiv1.Pod{}, errors.New("timed out waiting for pod to become ready")
		default:
			pods, _ := kc.GetPods(namespace)
			for _, pod := range pods {
				if strings.Contains(pod.Name, name) && pod.Status.Phase == apiv1.PodRunning {
					return pod, nil
				}
			}

			time.Sleep(5 * time.Second)
		}
	}
}
