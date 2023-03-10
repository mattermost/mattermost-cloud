package model

// ProvisioningParams represent configuration used during various provisioning operations.
type ProvisioningParams struct {
	S3StateStore            string
	AllowCIDRRangeList      []string
	VpnCIDRList             []string
	Owner                   string
	UseExistingAWSResources bool
	DeployMysqlOperator     bool
	DeployMinioOperator     bool
	NdotsValue              string
	PGBouncerConfig         *PGBouncerConfig
	SLOInstallationGroups   []string
	SLOEnterpriseGroups     []string
	EtcdManagerEnv          map[string]string
}
