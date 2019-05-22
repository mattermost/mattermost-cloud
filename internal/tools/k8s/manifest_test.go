package k8s

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	exampleServiceYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mattermost-operator`

	exampleBadYAML = `
badKey: v1
kind: ServiceAccount
metadata:`

	exampleMultiResourceYAML = `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: mysql-operator
  labels:
    app: mysql-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mysql-operator
  template:
    metadata:
      labels:
        app: mysql-operator
    spec:
      serviceAccountName: mysql-operator
      containers:
      - name: mysql-operator-controller
        imagePullPolicy: IfNotPresent
        image: iad.ocir.io/oracle/mysql-operator:0.3.0
        ports:
        - containerPort: 10254
        args:
          - --v=4
          - --mysql-agent-image=iad.ocir.io/oracle/mysql-agent
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: mysql-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: mysql-agent
subjects:
- kind: ServiceAccount
  name: mysql-agent
  namespace: mysql-operator
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: mysql-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind:  ClusterRole
  name: mysql-operator
subjects:
- kind: ServiceAccount
  name: mysql-operator
  namespace: mysql-operator
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: mysql-operator
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs:
    - get
    - list
    - patch
    - update
    - watch`
)

func TestCreate(t *testing.T) {
	testClient := newTestKubeClient()

	tempDir, err := ioutil.TempDir(".", "k8s-file-testing-")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	serviceYAML := filepath.Join(tempDir, "service.yaml")
	err = ioutil.WriteFile(serviceYAML, []byte(exampleServiceYAML), 0600)
	assert.NoError(t, err)

	multiYAML := filepath.Join(tempDir, "multi.yaml")
	err = ioutil.WriteFile(multiYAML, []byte(exampleMultiResourceYAML), 0600)
	assert.NoError(t, err)

	badYAML := filepath.Join(tempDir, "bad.yaml")
	err = ioutil.WriteFile(badYAML, []byte(exampleBadYAML), 0600)
	assert.NoError(t, err)

	namespace := "testing"

	t.Run("create from files", func(t *testing.T) {
		files := []ManifestFile{
			ManifestFile{
				Path:            serviceYAML,
				DeployNamespace: namespace,
			},
		}
		err := testClient.CreateFromFiles(files)
		assert.NoError(t, err)
	})
	t.Run("create from multi-resource file", func(t *testing.T) {
		files := []ManifestFile{
			ManifestFile{
				Path:            multiYAML,
				DeployNamespace: namespace,
			},
		}
		err := testClient.CreateFromFiles(files)
		assert.NoError(t, err)
	})
	t.Run("create with bad yaml format", func(t *testing.T) {
		files := []ManifestFile{
			ManifestFile{
				Path:            badYAML,
				DeployNamespace: namespace,
			},
		}
		err := testClient.CreateFromFiles(files)
		assert.Error(t, err)
	})
}

func TestBasename(t *testing.T) {
	var basenameTests = []struct {
		file     ManifestFile
		expected string
	}{
		{
			ManifestFile{Path: "/tmp/test/1/file1.yaml"},
			"file1.yaml",
		}, {
			ManifestFile{Path: "noDirectory.yaml"},
			"noDirectory.yaml",
		},
	}

	for _, tt := range basenameTests {
		t.Run(tt.file.Path, func(t *testing.T) {
			assert.Equal(t, tt.file.Basename(), tt.expected)
		})
	}
}
