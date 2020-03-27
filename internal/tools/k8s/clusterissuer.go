package k8s 

import (
	"encoding/json"

	"github.com/pkg/errors"

	v1alpha3 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (kc *KubeClient) createOrUpdateClusterIssuer(clusterissuer *v1alpha3.ClusterIssuer) (v1.Object, error){
	_, err := kc.JetStackClientset.CertmanagerV1alpha3().ClusterIssuers().Get(clusterissuer.GetName(), v1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.JetStackClientset.CertmanagerV1alpha3().ClusterIssuers().Create(clusterissuer)
	}
	patch, err := json.Marshal(clusterissuer)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal new Cluster Issuer definition")
	}

	return kc.JetStackClientset.CertmanagerV1alpha3().ClusterIssuers().Patch(clusterissuer.GetName(), types.MergePatchType, patch)
}
