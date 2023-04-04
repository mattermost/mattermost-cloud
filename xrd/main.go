// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"

	"github.com/mattermost/mattermost-cloud/internal/provisioner/crossplane"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// ctx := context.TODO()
	logger := logrus.New()
	namespace := "testing"
	obj := crossplane.EKS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: namespace,
			// Labels:      map[string]string{"test": "test"},
			// Annotations: ,
		},
		Spec: crossplane.EKSSpec{},
	}

	// obj := corev1.Pod{
	// 	// TypeMeta: metav1.TypeMeta{
	// 	// 	Kind:       "Pod",
	// 	// 	APIVersion: "v1",
	// 	// },
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      "test-nginx",
	// 		Namespace: namespace,
	// 	},
	// 	Spec: corev1.PodSpec{
	// 		Containers: []corev1.Container{{
	// 			Image: "nginx",
	// 		}},
	// 	},
	// }

	// cfg, err := rest.InClusterConfig()
	// if err != nil {
	// 	panic(err)
	// }

	// client, err := k8s.NewFromConfig(cfg, logger)
	// if err != nil {
	// 	panic(err)
	// }

	objBytes, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	client, err := k8s.NewFromFile("/Users/fmartingr/.kube/config_kind_provisioner", logger)
	if err != nil {
		panic(err)
	}

	req := client.Clientset.CoreV1().RESTClient().
		Post().
		Resource("cloud.mattermost.sio/mmK8ss").
		Namespace(obj.Namespace).
		Name(obj.Name).
		Body(objBytes)

	output, execErr := client.RemoteCommand("POST", req.URL())

	if execErr != nil {
		logger.WithError(execErr).Error("failed")
	} else {
		logger.Debugf(string(output))
	}

}
