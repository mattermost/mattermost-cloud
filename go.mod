module github.com/mattermost/mattermost-cloud

go 1.16

require (
	github.com/Masterminds/squirrel v1.4.0
	github.com/aws/aws-sdk-go v1.38.60
	github.com/blang/semver v3.5.1+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gosuri/uilive v0.0.4
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.8.0
	github.com/mattermost/awat v0.0.0-20210616202500-f0bdd4f43f90
	github.com/mattermost/mattermost-operator v1.15.0
	github.com/mattermost/rotator v0.1.2
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/olekukonko/tablewriter v0.0.5
	github.com/pborman/uuid v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.49.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.49.0
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/slok/sloth v0.6.0
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	github.com/vrischmann/envconfig v1.3.0
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/kube-aggregator v0.18.8
)
