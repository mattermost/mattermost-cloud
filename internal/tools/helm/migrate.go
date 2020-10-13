package helm

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func MigrateReleases(logger log.FieldLogger, kubeConfigPath string, release ...string) error {
	helm3, err := NewV3(logger.WithField("helm-version", "v3"))
	if err != nil {
		return errors.Wrap(err, "failed to initialize Helm 3 client")
	}

	// Dry run first
	err = helm2to3migrate(helm3, release, kubeConfigPath, true)
	if err != nil {
		return errors.Wrap(err, "failed to run Helm 2to3 migration dry run")
	}

	// Actual migration
	err = helm2to3migrate(helm3, release, kubeConfigPath, false)
	if err != nil {
		return errors.Wrap(err, "failed to migrate releases from Helm 2to3")
	}

	// Dry run first
	err = helm2to3Cleanup(helm3, release, kubeConfigPath, true)
	if err != nil {
		return errors.Wrap(err, "failed to run Helm 2to3 cleanup dry run")
	}

	// Actual cleanup
	err = helm2to3Cleanup(helm3, release, kubeConfigPath, false)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup releases from Helm 2")
	}

	return nil
}

func helm2to3migrate(helm3 *Cmd, releases []string, kubeConfigPath string, dryRun bool) error {
	for _, rel := range releases {
		err := convert(helm3, rel, kubeConfigPath, dryRun)
		if err != nil {
			return errors.Wrapf(err, "failed to convert '%s' release from Helm 2 to 3", rel)
		}
	}

	return nil
}

func helm2to3Cleanup(helm3 *Cmd, releases []string, kubeConfigPath string, dryRun bool) error {
	for _, rel := range releases {
		err := cleanup(helm3, rel, kubeConfigPath, dryRun)
		if err != nil {
			return errors.Wrapf(err, "failed to convert '%s' release from Helm 2 to 3", rel)
		}
	}

	return nil
}

func convert(helm3 *Cmd, release string, kubeConfigPath string, dryRun bool) error {
	args := []string{
		"2to3",
		"convert",
		"--kubeconfig", kubeConfigPath,
		release,
	}
	args = withDryRun(args, dryRun)

	return helm3.RunGenericCommand(args...)
}

func cleanup(helm3 *Cmd, release string, kubeConfigPath string, dryRun bool) error {
	args := []string{
		"2to3",
		"cleanup",
		"--kubeconfig", kubeConfigPath,
		"--name", release,
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
