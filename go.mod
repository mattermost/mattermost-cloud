module github.com/mattermost/mattermost-cloud

go 1.15

require (
	github.com/Masterminds/squirrel v1.4.0
	github.com/aws/aws-sdk-go v1.36.7
	github.com/blang/semver v3.5.1+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.4.4
	github.com/gorilla/mux v1.8.0
	github.com/gosuri/uilive v0.0.4
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.8.0
	github.com/mattermost/awat v0.0.0-20210428223656-1267d82207a0
	github.com/mattermost/mattermost-operator v1.13.0
	github.com/mattermost/rotator v0.1.2
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/olekukonko/tablewriter v0.0.4
	github.com/pborman/uuid v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.4.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.15
	k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery v0.18.15
	k8s.io/client-go v0.18.15
	k8s.io/kube-aggregator v0.18.8
)
