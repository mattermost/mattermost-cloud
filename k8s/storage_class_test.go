// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStorageClass(t *testing.T) {
	testClient := newTestKubeClient()
	storageClass := &v1beta1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "gp2"},
	}
	class := "gp2"

	t.Run("update storageclass", func(t *testing.T) {
		ctx := context.TODO()
		testClient.Clientset.StorageV1beta1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})
		result, err := testClient.UpdateStorageClassVolumeBindingMode(class)
		require.NoError(t, err)
		require.Equal(t, storageClass.GetName(), result.GetName())
	})

}
