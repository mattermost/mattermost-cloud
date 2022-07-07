// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import corev1 "k8s.io/api/core/v1"

// ClusterResources is a snapshot of a cluster's total and currently-used
// resources.
type ClusterResources struct {
	TotalNodeCount   int64
	SkippedNodeCount int64
	WorkerNodeCount  int64
	MilliTotalCPU    int64
	MilliUsedCPU     int64
	MilliTotalMemory int64
	MilliUsedMemory  int64
	TotalPodCount    int64
	UsedPodCount     int64
}

// CalculateCPUPercentUsed calculates the CPU usage percentage of a cluster with
// an optional additional load. Pass in 0 to calculate the current CPU usage of
// the cluster.
func (r *ClusterResources) CalculateCPUPercentUsed(additional int64) int {
	return int((float64(r.MilliUsedCPU+additional) / float64(r.MilliTotalCPU)) * 100)
}

// CalculateMemoryPercentUsed calculates the memory usage percentage of a
// cluster with an optional additional load. Pass in 0 to calculate the current
// memory usage of the cluster.
func (r *ClusterResources) CalculateMemoryPercentUsed(additional int64) int {
	return int((float64(r.MilliUsedMemory+additional) / float64(r.MilliTotalMemory)) * 100)
}

// CalculatePodCountPercentUsed calculates the pod count usage percentage of a
// cluster with an optional additional load. Pass in 0 to calculate the current
// pod count usage of the cluster.
func (r *ClusterResources) CalculatePodCountPercentUsed(additional int64) int {
	return int((float64(r.UsedPodCount+additional) / float64(r.TotalPodCount)) * 100)
}

// CalculateTotalPodMilliResourceRequests calculates the total CPU and memory
// milli resource requirements of a list of pods.
func CalculateTotalPodMilliResourceRequests(pods []corev1.Pod) (int64, int64) {
	var totalCPU, totalMemory int64
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			totalCPU += container.Resources.Requests.Cpu().MilliValue()
			totalMemory += container.Resources.Requests.Memory().MilliValue()
		}
	}

	return totalCPU, totalMemory
}
