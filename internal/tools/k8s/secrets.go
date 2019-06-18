package k8s

import (
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateSecret(namespace string, secret *corev1.Secret) (metav1.Object, error) {
	_, err := kc.Clientset.CoreV1().Secrets(namespace).Get(secret.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.CoreV1().Secrets(namespace).Create(secret)
	}

	return kc.Clientset.CoreV1().Secrets(namespace).Update(secret)
}

// CreateSecret creates kubernetes secret.
func (kc *KubeClient) CreateSecret(namespaceName, secretName, secretKey, secretValue string) (*corev1.Secret, error) {
	// Check if secret exists first.
	svc, err := kc.Clientset.CoreV1().Secrets(namespaceName).Get(secretName, metav1.GetOptions{})
	if err == nil {
		return svc, nil
	}

	svcSpec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
		},
		StringData: map[string]string{
			secretKey: secretValue,
		},
	}
	return kc.Clientset.CoreV1().Secrets(namespaceName).Create(svcSpec)
}

// GetSecrets returns a list of kubernetes secret objects if they exist.
//
// Any errors will be returned immediately and the remaining secrets will be
// skipped.
func (kc *KubeClient) GetSecrets(nameSpace string, secretsNames []string) ([]*corev1.Secret, error) {
	secrets := []*corev1.Secret{}
	for _, secretsName := range secretsNames {
		secret, err := kc.Clientset.CoreV1().Secrets(nameSpace).Get(secretsName, metav1.GetOptions{})
		if err != nil {
			return secrets, err
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}
