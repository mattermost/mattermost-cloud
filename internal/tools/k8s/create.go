package k8s

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	mmv1alpha1 "github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg/apis/mattermost.com/v1alpha1"
	mmclientV1alpha1 "github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg/clientset/versioned"
	mattermostscheme "github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg/clientset/versioned/scheme"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	appsbetav1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apixv1beta1scheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

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

func (kc *KubeClient) createDeploymentV1(namespace string, deployment *appsv1.Deployment) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.AppsV1().Deployments(namespace)

	result, err := client.Create(deployment)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createCustomResourceDefinition(crd *apixv1beta1.CustomResourceDefinition) (metav1.Object, error) {
	clientset, err := apixv1beta1client.NewForConfig(kc.config)
	if err != nil {
		return nil, err
	}
	client := clientset.CustomResourceDefinitions()

	result, err := client.Create(crd)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterInstallation(namespace string, ci *mmv1alpha1.ClusterInstallation) (metav1.Object, error) {
	clientset, err := mmclientV1alpha1.NewForConfig(kc.config)
	if err != nil {
		return nil, err
	}
	client := clientset.ExampleV1alpha1().ClusterInstallations(namespace)

	result, err := client.Create(ci)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createDeploymentBetaV1(namespace string, deployment *appsbetav1.Deployment) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.AppsV1beta1().Deployments(namespace)

	result, err := client.Create(deployment)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createServiceAccount(namespace string, account *apiv1.ServiceAccount) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.CoreV1().ServiceAccounts(namespace)

	result, err := client.Create(account)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createRoleBindingV1(namespace string, binding *rbacv1.RoleBinding) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.RbacV1().RoleBindings(namespace)

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createRoleBindingBetaV1(namespace string, binding *rbacbetav1.RoleBinding) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.RbacV1beta1().RoleBindings(namespace)

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterRoleV1(account *rbacv1.ClusterRole) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.RbacV1().ClusterRoles()

	result, err := client.Create(account)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterRoleBetaV1(account *rbacbetav1.ClusterRole) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.RbacV1beta1().ClusterRoles()

	result, err := client.Create(account)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterRoleBindingV1(binding *rbacv1.ClusterRoleBinding) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.RbacV1().ClusterRoleBindings()

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterRoleBindingBetaV1(binding *rbacbetav1.ClusterRoleBinding) (metav1.Object, error) {
	clientset, err := kc.getKubeConfigClientset()
	if err != nil {
		return nil, err
	}
	client := clientset.RbacV1beta1().ClusterRoleBindings()

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}
