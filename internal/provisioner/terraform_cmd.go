package provisioner

import log "github.com/sirupsen/logrus"

type terraformFactoryFunc func(outputDir string, logger log.FieldLogger) TerraformCmd

// TerraformCmd describes the interface required by the provisioner to interact with terraform.
type TerraformCmd interface {
	Init() error
	Apply() error
	ApplyTarget(target string) error
	Output(variable string) (string, error)
	Destroy() error
	Close() error
}
