module github.com/mattermost/mattermost-cloud

go 1.14

require (
	github.com/Masterminds/squirrel v1.4.0
	github.com/aws/aws-sdk-go v1.31.9
	github.com/blang/semver v3.5.1+incompatible
	github.com/emicklei/go-restful v2.11.2+incompatible // indirect
	github.com/go-openapi/spec v0.19.6 // indirect
	github.com/go-openapi/swag v0.19.7 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.4.3
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.6.0
	github.com/mattermost/mattermost-operator v1.4.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.0
	golang.org/x/sys v0.0.0-20200212091648-12a6c2dcc1e4 // indirect
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.17.3
	k8s.io/utils v0.0.0-20200124190032-861946025e34 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

// Pinned to kubernetes-1.16.10
replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.2.0
	k8s.io/api => k8s.io/api v0.16.10
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.10
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.10
	k8s.io/apiserver => k8s.io/apiserver v0.16.10
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.10
	k8s.io/client-go => k8s.io/client-go v0.16.10
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.10
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.10
	k8s.io/code-generator => k8s.io/code-generator v0.16.8-beta.0
	k8s.io/component-base => k8s.io/component-base v0.16.10
	k8s.io/cri-api => k8s.io/cri-api v0.16.8-beta.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.10
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.10
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.10
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.10
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.10
	k8s.io/kubectl => k8s.io/kubectl v0.16.10
	k8s.io/kubelet => k8s.io/kubelet v0.16.10
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.16.10
	k8s.io/metrics => k8s.io/metrics v0.16.10
	k8s.io/node-api => k8s.io/node-api v0.16.10
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.16.10
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.16.10
	k8s.io/sample-controller => k8s.io/sample-controller v0.16.10
)
