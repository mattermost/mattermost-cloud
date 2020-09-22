module github.com/mattermost/mattermost-cloud

go 1.14

require (
	github.com/Masterminds/squirrel v1.4.0
	github.com/aws/aws-sdk-go v1.34.26
	github.com/blang/semver v3.5.1+incompatible
	github.com/emicklei/go-restful v2.11.2+incompatible // indirect
	github.com/go-openapi/swag v0.19.9 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.4.4
	github.com/gorilla/mux v1.7.4
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.8.0
	github.com/mattermost/mattermost-operator v1.7.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/pborman/uuid v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.18.9
)

// Pinned to kubernetes-0.18.9
replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	k8s.io/client-go => k8s.io/client-go v0.18.9
)
