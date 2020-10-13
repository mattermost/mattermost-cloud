package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// MigrateRelease migrates Helm release from Helm 2 to Helm 3
func MigrateRelease(logger log.FieldLogger, kubeConfigPath string, release string) error {
	helm3, err := helm.NewV3(logger.WithField("helm-version", "v3"))
	if err != nil {
		return errors.Wrap(err, "failed to initialize Helm 3 client")
	}

	// Dry run first
	err = convert(helm3, release, kubeConfigPath, true)
	if err != nil {
		return errors.Wrap(err, "failed to run Helm 2to3 migration dry run")
	}

	// Actual migration
	err = convert(helm3, release, kubeConfigPath, false)
	if err != nil {
		return errors.Wrap(err, "failed to migrate releases from Helm 2to3")
	}

	// Dry run first
	err = cleanup(helm3, release, kubeConfigPath, true)
	if err != nil {
		return errors.Wrap(err, "failed to run Helm 2to3 cleanup dry run")
	}

	// Actual cleanup
	err = cleanup(helm3, release, kubeConfigPath, false)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup releases from Helm 2")
	}

	return nil
}

// CleanupAll cleans up all Helm 2 releases together with Tiller
func CleanupAll(logger log.FieldLogger, kubeConfigPath string) error {
	helm3, err := helm.NewV3(logger.WithField("helm-version", "v3"))
	if err != nil {
		return errors.Wrap(err, "failed to initialize Helm 3 client")
	}

	// Dry run
	err = cleanupAll(helm3, kubeConfigPath, true)
	if err != nil {
		return errors.Wrap(err, "failed to dry run of cleanup remaining Helm 2 resources")
	}

	// Actual cleanup
	err = cleanupAll(helm3, kubeConfigPath, false)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup remaining Helm 2 resources")
	}

	return nil
}


func convert(helm3 *helm.Cmd, release string, kubeConfigPath string, dryRun bool) error {
	args := []string{
		"2to3",
		"convert",
		"--kubeconfig", kubeConfigPath,
		release,
	}
	args = withDryRun(args, dryRun)

	return helm3.RunGenericCommand(args...)
}

func cleanup(helm3 *helm.Cmd, release string, kubeConfigPath string, dryRun bool) error {
	args := []string{
		"2to3",
		"cleanup",
		"--skip-confirmation",
		"--kubeconfig", kubeConfigPath,
		"--name", release,
	}
	args = withDryRun(args, dryRun)

	return helm3.RunGenericCommand(args...)
}

func cleanupAll(helm3 *helm.Cmd, kubeConfigPath string, dryRun bool) error {
	args := []string{
		"2to3",
		"cleanup",
		"--skip-confirmation",
		"--kubeconfig", kubeConfigPath,
	}
	args = withDryRun(args, dryRun)

	return helm3.RunGenericCommand(args...)
}

func withDryRun(args []string, dryRun bool) []string {
	if dryRun {
		return append(args, "--dry-run")
	}
	return args
}
