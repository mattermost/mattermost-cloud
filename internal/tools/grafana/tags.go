package grafana

import "fmt"

const (
	provisionerTag      = "provisioner"
	clusterProvisionTag = "cluster-provision"
	clusterUpgradeTag   = "cluster-upgrade"
	clusterResizeTag    = "cluster-resize"
)

func newKVTag(k, v string) string {
	return fmt.Sprintf("%s:%s", k, v)
}
