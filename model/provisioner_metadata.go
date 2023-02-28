package model

type ProvisionerMetadata struct {
	Name             string
	Version          string
	AMI              string
	NodeInstanceType string
	NodeMinCount     int64
	NodeMaxCount     int64
	MaxPodsPerNode   int64
	VPC              string
	Networking       string
}
