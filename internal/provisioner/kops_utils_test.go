package provisioner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGrossReplace(t *testing.T) {
	testManifest := `
apiVersion: kops.k8s.io/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: "2020-03-19T20:33:45Z"
  labels:
    kops.k8s.io/cluster: 1nx98f8ykbbz9ern94reuodqpe-kops.k8s.local
  name: nodes
spec:
  additionalSecurityGroups:
  - sg-08bc68b2c11d412fc
  image: kope.io/k8s-1.15-debian-stretch-amd64-hvm-ebs-2020-01-17
  machineType: m5.large
  maxSize: 26
  minSize: 26
  nodeLabels:
    kops.k8s.io/instancegroup: nodes
  role: Node
  subnets:
  - us-east-1a
  - us-east-1b
  - us-east-1c
  - us-east-1d
  - us-east-1e
`

	expectedManifest := `
apiVersion: kops.k8s.io/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: "2020-03-19T20:33:45Z"
  labels:
    kops.k8s.io/cluster: 1nx98f8ykbbz9ern94reuodqpe-kops.k8s.local
  name: nodes
spec:
  additionalSecurityGroups:
  - sg-08bc68b2c11d412fc
  image: kope.io/k8s-1.15-debian-stretch-amd64-hvm-ebs-2020-01-17
  machineType: raspberry.pi
  maxSize: 1337
  minSize: 1337
  nodeLabels:
    kops.k8s.io/instancegroup: nodes
  role: Node
  subnets:
  - us-east-1a
  - us-east-1b
  - us-east-1c
  - us-east-1d
  - us-east-1e
`
	t.Run("valid replace", func(t *testing.T) {
		replaced, err := grossKopsReplaceSize(testManifest, "raspberry.pi", "1337", "1337")
		assert.NoError(t, err)
		assert.Equal(t, expectedManifest, replaced)
	})

	testManifest = `
apiVersion: kops.k8s.io/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: "2020-03-19T20:33:45Z"
  labels:
    kops.k8s.io/cluster: 1nx98f8ykbbz9ern94reuodqpe-kops.k8s.local
  name: nodes
spec:
  additionalSecurityGroups:
  - sg-08bc68b2c11d412fc
  image: kope.io/k8s-1.15-debian-stretch-amd64-hvm-ebs-2020-01-17
  machineType: m5.large
  minSize: 26
  nodeLabels:
    kops.k8s.io/instancegroup: nodes
  role: Node
  subnets:
  - us-east-1a
  - us-east-1b
  - us-east-1c
  - us-east-1d
  - us-east-1e
`

	t.Run("no maxSize", func(t *testing.T) {
		replaced, err := grossKopsReplaceSize(testManifest, "raspberry.pi", "1337", "1337")
		assert.Error(t, err)
		assert.Empty(t, replaced)
	})

	testManifest = `
apiVersion: kops.k8s.io/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: "2020-03-19T20:33:45Z"
  labels:
    kops.k8s.io/cluster: 1nx98f8ykbbz9ern94reuodqpe-kops.k8s.local
  name: nodes
spec:
  additionalSecurityGroups:
  - sg-08bc68b2c11d412fc
  image: kope.io/k8s-1.15-debian-stretch-amd64-hvm-ebs-2020-01-17
  machineType: m5.large
  maxSize: 26
  nodeLabels:
    kops.k8s.io/instancegroup: nodes
  role: Node
  subnets:
  - us-east-1a
  - us-east-1b
  - us-east-1c
  - us-east-1d
  - us-east-1e
`

	t.Run("no minSize", func(t *testing.T) {
		replaced, err := grossKopsReplaceSize(testManifest, "raspberry.pi", "1337", "1337")
		assert.Error(t, err)
		assert.Empty(t, replaced)
	})

	testManifest = `
apiVersion: kops.k8s.io/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: "2020-03-19T20:33:45Z"
  labels:
    kops.k8s.io/cluster: 1nx98f8ykbbz9ern94reuodqpe-kops.k8s.local
  name: nodes
spec:
  additionalSecurityGroups:
  - sg-08bc68b2c11d412fc
  image: kope.io/k8s-1.15-debian-stretch-amd64-hvm-ebs-2020-01-17
  maxSize: 26
  minSize: 26
  nodeLabels:
    kops.k8s.io/instancegroup: nodes
  role: Node
  subnets:
  - us-east-1a
  - us-east-1b
  - us-east-1c
  - us-east-1d
  - us-east-1e
`

	t.Run("no machineType", func(t *testing.T) {
		replaced, err := grossKopsReplaceSize(testManifest, "raspberry.pi", "1337", "1337")
		assert.Error(t, err)
		assert.Empty(t, replaced)
	})
}
