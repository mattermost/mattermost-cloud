package k8s

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	mattermostscheme "github.com/mattermost/mattermost-operator/pkg/client/clientset/versioned/scheme"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	appsbetav1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apixv1beta1scheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// ManifestFile is a file containing kubernetes resources.
type ManifestFile struct {
	Path            string
	DeployNamespace string
}

// Basename returns the base filename of the manifest file.
func (f *ManifestFile) Basename() string {
	return path.Base(f.Path)
}

// CreateFromFiles will create Kubernetes resources from the provided manifest
// files.
func (kc *KubeClient) CreateFromFiles(files []ManifestFile) error {
	for _, f := range files {
		err := kc.CreateFromFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateFromFile will create the Kubernetes resources in the provided file.
//
// The current behavior leads to the create being attempted on all resources in
// the provided file. An error is returned if any of the create actions failed.
// This process equates to running `kubectl create -f FILENAME`.
func (kc *KubeClient) CreateFromFile(file ManifestFile) error {
	data, err := ioutil.ReadFile(file.Path)
	if err != nil {
		return err
	}

	apixv1beta1scheme.AddToScheme(scheme.Scheme)
	mattermostscheme.AddToScheme(scheme.Scheme)

	logger := kc.logger.WithFields(log.Fields{
		"file": file.Basename(),
	})

	var failures int
	resources := bytes.Split(data, []byte("---"))
	for _, resource := range resources {
		if len(resource) == 0 {
			continue
		}
		decode := scheme.Codecs.UniversalDeserializer().Decode

		obj, _, err := decode(resource, nil, nil)
		if err != nil {
			logger.Error(errors.Wrap(err, "unable to decode k8s resource"))
			failures++
			continue
		}

		result, err := kc.createFileResource(file.DeployNamespace, obj)
		if err != nil {
			logger.Error(errors.Wrap(err, "unable to create k8s resource"))
			failures++
			continue
		}

		logger.Infof("Resource %q created!", result.GetName())
	}

	if failures > 0 {
		return fmt.Errorf("encountered %d create failures", failures)
	}

	return nil
}

func (kc *KubeClient) createFileResource(deployNamespace string, obj interface{}) (metav1.Object, error) {
	switch o := obj.(type) {
	case *apiv1.ServiceAccount:
		return kc.createServiceAccount(deployNamespace, obj.(*apiv1.ServiceAccount))
	case *appsv1.Deployment:
		return kc.createDeploymentV1(deployNamespace, obj.(*appsv1.Deployment))
	case *appsbetav1.Deployment:
		return kc.createDeploymentBetaV1(deployNamespace, obj.(*appsbetav1.Deployment))
	case *rbacv1.RoleBinding:
		return kc.createRoleBindingV1(deployNamespace, obj.(*rbacv1.RoleBinding))
	case *rbacbetav1.RoleBinding:
		return kc.createRoleBindingBetaV1(deployNamespace, obj.(*rbacbetav1.RoleBinding))
	case *rbacv1.ClusterRole:
		return kc.createClusterRoleV1(obj.(*rbacv1.ClusterRole))
	case *rbacbetav1.ClusterRole:
		return kc.createClusterRoleBetaV1(obj.(*rbacbetav1.ClusterRole))
	case *rbacv1.ClusterRoleBinding:
		return kc.createClusterRoleBindingV1(obj.(*rbacv1.ClusterRoleBinding))
	case *rbacbetav1.ClusterRoleBinding:
		return kc.createClusterRoleBindingBetaV1(obj.(*rbacbetav1.ClusterRoleBinding))
	case *apixv1beta1.CustomResourceDefinition:
		return kc.createCustomResourceDefinition(obj.(*apixv1beta1.CustomResourceDefinition))
	case *mmv1alpha1.ClusterInstallation:
		return kc.createClusterInstallation(deployNamespace, obj.(*mmv1alpha1.ClusterInstallation))
	default:
		return nil, fmt.Errorf("Error: unsupported k8s manifest type %T", o)
	}
}
