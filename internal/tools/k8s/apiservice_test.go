// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

func TestAPIRegistrationV1Beta1(t *testing.T) {
	testClient := newTestKubeClient()
	apiReg := &apiregistrationv1beta1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: "test-apiregistration"},
	}

	t.Run("create APIRegistration", func(t *testing.T) {
		result, err := testClient.createOrUpdateAPIServer(apiReg)
		require.NoError(t, err)
		require.Equal(t, apiReg.GetName(), result.GetName())
	})

	t.Run("create duplicate APIRegistration", func(t *testing.T) {
		result, err := testClient.createOrUpdateAPIServer(apiReg)
		require.NoError(t, err)
		require.Equal(t, apiReg.GetName(), result.GetName())
	})
}
