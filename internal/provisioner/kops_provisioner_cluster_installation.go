package provisioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// CreateClusterInstallation creates a Mattermost installation within the given cluster.
func (provisioner *KopsProvisioner) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation, awsClient aws.AWS) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Creating cluster installation")

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
	if err != nil {
		return errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	_, err = k8sClient.CreateOrUpdateNamespace(clusterInstallation.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create namespace %s", clusterInstallation.Namespace)
	}

	installationName := makeClusterInstallationName(clusterInstallation)

	file := k8s.ManifestFile{
		Path:            "manifests/network-policies/mm-installation-netpol.yaml",
		DeployNamespace: clusterInstallation.Namespace,
	}
	err = k8sClient.CreateFromFile(file, installationName)
	if err != nil {
		return errors.Wrapf(err, "failed to create network policy %s", clusterInstallation.Namespace)
	}

	mattermostInstallation := &mmv1alpha1.ClusterInstallation{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterInstallation",
			APIVersion: "mattermost.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      installationName,
			Namespace: clusterInstallation.Namespace,
			Labels: map[string]string{
				"installation":         installation.ID,
				"cluster-installation": clusterInstallation.ID,
			},
		},
		Spec: mmv1alpha1.ClusterInstallationSpec{
			Size:          installation.Size,
			Version:       translateMattermostVersion(installation.Version),
			Image:         installation.Image,
			IngressName:   installation.DNS,
			MattermostEnv: installation.MattermostEnv.ToEnvList(),
			UseIngressTLS: false,
			IngressAnnotations: map[string]string{
				"kubernetes.io/ingress.class":                          "nginx",
				"kubernetes.io/tls-acme":                               "true",
				"nginx.ingress.kubernetes.io/proxy-buffering":          "on",
				"nginx.ingress.kubernetes.io/proxy-body-size":          "100m",
				"nginx.ingress.kubernetes.io/proxy-send-timeout":       "600",
				"nginx.ingress.kubernetes.io/proxy-read-timeout":       "600",
				"nginx.ingress.kubernetes.io/proxy-max-temp-file-size": "0",
				"nginx.ingress.kubernetes.io/ssl-redirect":             "true",
				"nginx.ingress.kubernetes.io/configuration-snippet": `
				  proxy_force_ranges on;
				  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
				  proxy_cache mattermost_cache;
				  proxy_cache_revalidate on;
				  proxy_cache_min_uses 2;
				  proxy_cache_use_stale timeout;
				  proxy_cache_lock on;
				  proxy_cache_key "$host$request_uri$cookie_user";`,
				"nginx.org/server-snippets": "gzip on;",
			},
		},
	}

	if installation.License != "" {
		licenseSecretName := fmt.Sprintf("%s-license", makeClusterInstallationName(clusterInstallation))
		licenseSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: licenseSecretName,
			},
			StringData: map[string]string{"license": installation.License},
		}

		_, err = k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, licenseSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create the license secret %s/%s", clusterInstallation.Namespace, licenseSecretName)
		}

		mattermostInstallation.Spec.MattermostLicenseSecret = licenseSecretName
		logger.Debug("Cluster installation configured with a Mattermost license")
	}

	databaseSpec, databaseSecret, err := provisioner.resourceUtil.GetDatabase(installation).GenerateDatabaseSpecAndSecret(provisioner.store, logger)
	if err != nil {
		return err
	}

	if databaseSpec != nil {
		_, err = k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, databaseSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create the database secret %s/%s", clusterInstallation.Namespace, databaseSecret.Name)
		}
		mattermostInstallation.Spec.Database = *databaseSpec
	}

	filestoreSpec, filestoreSecret, err := provisioner.resourceUtil.GetFilestore(installation).GenerateFilestoreSpecAndSecret(logger)
	if err != nil {
		return err
	}

	if filestoreSecret != nil {
		_, err = k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, filestoreSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create the filestore secret %s/%s", clusterInstallation.Namespace, filestoreSecret.Name)
		}
		mattermostInstallation.Spec.Minio = *filestoreSpec
	}

	_, err = k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Create(mattermostInstallation)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation")
	}

	logger.Info("Successfully created cluster installation")

	return nil
}

// HibernateClusterInstallation updates a cluster installation to consume fewer
// resources.
func (provisioner *KopsProvisioner) HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
	if err != nil {
		return errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	name := makeClusterInstallationName(clusterInstallation)
	cr, err := k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	// Hibernation is currently considered changing the Mattermost app deployment
	// to 0 replicas in the pod. i.e. Scale down to no Mattermost apps running.
	// The current way to do this is to set a negative replica count in the
	// k8s custom resource.
	// TODO: enhance hibernation to include database and/or filestore.
	cr.Spec.Replicas = -1

	_, err = k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Update(cr)
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}

	logger.Info("Updated cluster installation")

	return nil
}

// UpdateClusterInstallation updates the cluster installation spec to match the
// installation specification.
func (provisioner *KopsProvisioner) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	if cluster.ProvisionerMetadataKops.Name == "" {
		logger.Infof("Cluster %s has no name, assuming cluster installation never existed.", cluster.ID)
		return nil
	}

	err = kops.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
	if err != nil {
		return errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	name := makeClusterInstallationName(clusterInstallation)

	file := k8s.ManifestFile{
		Path:            "manifests/network-policies/mm-installation-netpol.yaml",
		DeployNamespace: clusterInstallation.Namespace,
	}
	err = k8sClient.CreateFromFile(file, name)
	if err != nil {
		return errors.Wrapf(err, "failed to create network policy %s", clusterInstallation.Namespace)
	}

	cr, err := k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	logger.WithField("status", fmt.Sprintf("%+v", cr.Status)).Debug("Got cluster installation")

	version := translateMattermostVersion(installation.Version)
	if cr.Spec.Version == version {
		logger.Debugf("Cluster installation already on version %s", version)
	} else {
		logger.Debugf("Cluster installation version updated from %s to %s", cr.Spec.Version, installation.Version)
		cr.Spec.Version = version
	}

	if cr.Spec.Image == installation.Image {
		logger.Debugf("Cluster installation already on image %s", installation.Image)
	} else {
		logger.Debugf("Cluster installation image updated from %s to %s", cr.Spec.Image, installation.Image)
		cr.Spec.Image = installation.Image
	}

	// A few notes on installation sizing changes:
	//  - Resource and replica changes are currently targeting the Mattermost
	//    app pod only. Other pods such as those managed by the MinIO and
	//    MySQL operator won't be updated with the current Mattermost
	//    Operator logic. This is intentional as those apps can have replica
	//    scaling issues.
	//  - Resizing currently ignores the installation scheduling algorithm.
	//    There is no good interface to determine if the new installation
	//    size will safely fit on the cluster. This could, in theory, be done
	//    when the size request change comes in on the API, but would require
	//    new scheduling logic. For now, take care when resizing.
	//    TODO: address these issue.
	if cr.Spec.Size == installation.Size {
		logger.Debugf("Cluster installation already on size %s", installation.Size)
	} else {
		logger.Debugf("Cluster installation size updated from %s to %s", cr.Spec.Size, installation.Size)
		cr.Spec.Size = installation.Size
	}

	sizeTemplate, err := mmv1alpha1.GetClusterSize(installation.Size)
	if err != nil {
		return errors.Wrap(err, "failed to get size requirements")
	}
	if cr.Spec.Replicas == sizeTemplate.App.Replicas {
		logger.Debugf("Cluster installation already has %d replicas", sizeTemplate.App.Replicas)
	} else {
		logger.Debugf("Cluster installation replicas updated from %d to %d", cr.Spec.Replicas, sizeTemplate.App.Replicas)
		cr.Spec.Replicas = sizeTemplate.App.Replicas
	}
	// Always ensure resources match
	cr.Spec.Resources = sizeTemplate.App.Resources

	cr.Spec.MattermostLicenseSecret = ""
	secretName := fmt.Sprintf("%s-license", name)
	if installation.License != "" {
		secretSpec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: clusterInstallation.Namespace,
			},
			StringData: map[string]string{
				"license": installation.License,
			},
		}
		_, err = k8sClient.CreateOrUpdateSecret(clusterInstallation.Namespace, secretSpec)
		if err != nil {
			return errors.Wrapf(err, "failed to create the license secret %s/%s", clusterInstallation.Namespace, secretName)
		}
		cr.Spec.MattermostLicenseSecret = secretName
	} else {
		err = k8sClient.Clientset.CoreV1().Secrets(clusterInstallation.Namespace).Delete(secretName, nil)
		if k8sErrors.IsNotFound(err) {
			logger.Infof("Secret %s/%s not found. Maybe the license was not set for this installation or was already deleted", clusterInstallation.Namespace, secretName)
		}
	}

	cr.Spec.MattermostEnv = installation.MattermostEnv.ToEnvList()

	_, err = k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Update(cr)
	if err != nil {
		return errors.Wrapf(err, "failed to update cluster installation %s", clusterInstallation.ID)
	}

	logger.Info("Updated cluster installation")

	return nil
}

// DeleteClusterInstallation deletes a Mattermost installation within the given cluster.
func (provisioner *KopsProvisioner) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	if cluster.ProvisionerMetadataKops.Name == "" {
		logger.Infof("Cluster %s has no name, assuming cluster installation never existed.", cluster.ID)
		return nil
	}

	err = kops.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
	if err != nil {
		return errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	name := makeClusterInstallationName(clusterInstallation)

	err = k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Delete(name, nil)
	if k8sErrors.IsNotFound(err) {
		logger.Warnf("Cluster installation %s not found, assuming already deleted", name)
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete cluster installation %s", clusterInstallation.ID)
	}

	if installation.License != "" {
		secretName := fmt.Sprintf("%s-license", makeClusterInstallationName(clusterInstallation))
		err = k8sClient.Clientset.CoreV1().Secrets(clusterInstallation.Namespace).Delete(secretName, nil)
		if k8sErrors.IsNotFound(err) {
			logger.Warnf("Secret %s/%s not found, assuming already deleted", clusterInstallation.Namespace, secretName)
		} else if err != nil {
			return errors.Wrapf(err, "failed to delete secret %s/%s", clusterInstallation.Namespace, secretName)
		}
	}

	err = k8sClient.Clientset.CoreV1().Namespaces().Delete(clusterInstallation.Namespace, &metav1.DeleteOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Warnf("Namespace %s not found, assuming already deleted", clusterInstallation.Namespace)
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete namespace %s", clusterInstallation.Namespace)
	}

	logger.Info("Successfully deleted cluster installation")

	return nil
}

// GetClusterInstallationResource gets the cluster installation resource from
// the kubernetes API.
func (provisioner *KopsProvisioner) GetClusterInstallationResource(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (*mmv1alpha1.ClusterInstallation, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	if cluster.ProvisionerMetadataKops.Name == "" {
		logger.Infof("Cluster %s has no name, assuming cluster installation never existed.", cluster.ID)
		return nil, nil
	}

	err = kops.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return nil, err
	}

	name := makeClusterInstallationName(clusterInstallation)

	cr, err := k8sClient.MattermostClientset.MattermostV1alpha1().ClusterInstallations(clusterInstallation.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return cr, errors.Wrapf(err, "failed to get cluster installation %s", clusterInstallation.ID)
	}

	logger.WithField("status", fmt.Sprintf("%+v", cr.Status)).Debug("Got cluster installation")

	return cr, nil
}

// ExecMattermostCLI invokes the Mattermost CLI for the given cluster installation with the given args.
func (provisioner *KopsProvisioner) ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	return provisioner.execCLI(cluster, clusterInstallation, append([]string{"./bin/mattermost"}, args...)...)
}

func (provisioner *KopsProvisioner) execCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct k8s client")
	}

	podList, err := k8sClient.Clientset.CoreV1().Pods(clusterInstallation.Namespace).List(metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query mattermost pods")
	}

	// In the future, we'd ideally just spin our own container on demand, allowing
	// configuration changes even if the pods are failing to start the server. For now,
	// we find the first pod running Mattermost, and pick the first container therein.

	if len(podList.Items) == 0 {
		return nil, errors.New("failed to find mattermost pods on which to exec")
	}

	pod := podList.Items[0]
	if len(pod.Spec.Containers) == 0 {
		return nil, errors.Errorf("failed to find containers in pod %s", pod.Name)
	}

	container := pod.Spec.Containers[0]
	logger.Debugf("Executing `%s` on pod %s, container %s, running image %s", strings.Join(args, " "), pod.Name, container.Name, container.Image)

	execRequest := k8sClient.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(clusterInstallation.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container.Name,
			Command:   args,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	now := time.Now()
	output, err := k8sClient.RemoteCommand("POST", execRequest.URL())

	logger.Debugf("Command `%s` on pod %s finished in %.0f seconds", strings.Join(args, " "), pod.Name, time.Since(now).Seconds())

	return output, err
}
