// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

// TODO: this file can be removed after full migration to Helm 3

import (
	"context"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

type helm2ReleaseJSON struct {
	Name       string `json:"Name"`
	Revision   int    `json:"Revision"`
	Updated    string `json:"Updated"`
	Status     string `json:"Status"`
	Chart      string `json:"Chart"`
	AppVersion string `json:"AppVersion"`
	Namespace  string `json:"Namespace"`
}

func (r helm2ReleaseJSON) toHelmRelease() helmReleaseJSON {
	return helmReleaseJSON{
		Name:       r.Name,
		Revision:   strconv.Itoa(r.Revision),
		Updated:    r.Updated,
		Status:     r.Status,
		Chart:      r.Chart,
		AppVersion: r.AppVersion,
		Namespace:  r.Namespace,
	}
}

// Helm2ListOutput is a struct for holding the unmarshaled
// representation of the output from helm list --output json (for Helm 2)
type Helm2ListOutput struct {
	Releases []helm2ReleaseJSON `json:"Releases"`
}

func (l Helm2ListOutput) asListOutput() *HelmListOutput {
	out := make([]helmReleaseJSON, 0, len(l.Releases))

	for _, rel := range l.Releases {
		out = append(out, rel.toHelmRelease())
	}
	list := HelmListOutput(out)
	return &list
}

// helmSetup is used for the initial setup of Helm in cluster.
func (um *HelmUtilsManager) helmSetup(logger log.FieldLogger, kops *kops.Cmd) error {
	k8sClient, err := k8s.NewFromFile(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	logger.Info("Creating Tiller service account")
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "tiller"},
	}

	ctx := context.TODO()
	_, err = k8sClient.Clientset.CoreV1().ServiceAccounts("kube-system").Get(ctx, "tiller", metav1.GetOptions{})
	if err != nil {
		// need to create cluster role bindings for Tiller since they couldn't be found

		_, err = k8sClient.Clientset.CoreV1().ServiceAccounts("kube-system").Create(ctx, serviceAccount, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to set up Tiller service account for Helm")
		}

		logger.Info("Creating Tiller cluster role bind")
		roleBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "tiller-cluster-rule"},
			Subjects: []rbacv1.Subject{
				{Kind: "ServiceAccount", Name: "tiller", Namespace: "kube-system"},
			},
			RoleRef: rbacv1.RoleRef{Kind: "ClusterRole", Name: "cluster-admin"},
		}

		_, err = k8sClient.Clientset.RbacV1().ClusterRoleBindings().Create(ctx, roleBinding, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster role bindings")
		}
	}

	err = um.helmInit(logger, kops)
	if err != nil {
		return err
	}

	return nil
}

// helmInit calls helm init and doesn't do anything fancy
func (um *HelmUtilsManager) helmInit(logger log.FieldLogger, kops *kops.Cmd) error {
	logger.Info("Upgrading Helm")
	helmClient, err := um.helmClientProvider(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand("--debug", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--service-account", "tiller", "--upgrade")
	if err != nil {
		return errors.Wrap(err, "failed to upgrade helm")
	}

	return nil
}
