// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	awsTools "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const hiddenLicense = "hidden (--hide-license=true)"

func newCmdInstallation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "installation",
		Short: "Manipulate installations managed by the provisioning server.",
	}

	setClusterFlags(cmd)

	cmd.AddCommand(newCmdInstallationCreate())
	cmd.AddCommand(newCmdInstallationUpdate())
	cmd.AddCommand(newCmdInstallationDelete())
	cmd.AddCommand(newCmdInstallationVolumeCreate())
	cmd.AddCommand(newCmdInstallationVolumeUpdate())
	cmd.AddCommand(newCmdInstallationVolumeDelete())
	cmd.AddCommand(newCmdInstallationUpdateDeletion())
	cmd.AddCommand(newCmdInstallationCancelDeletion())
	cmd.AddCommand(newCmdInstallationScheduleDeletion())
	cmd.AddCommand(newCmdInstallationHibernate())
	cmd.AddCommand(newCmdInstallationWakeup())
	cmd.AddCommand(newCmdInstallationGet())
	cmd.AddCommand(newCmdInstallationList())
	cmd.AddCommand(newCmdInstallationGetStatuses())
	cmd.AddCommand(newCmdInstallationShowStateReport())
	cmd.AddCommand(newCmdInstallationRecovery())
	cmd.AddCommand(newCmdInstallationDeploymentReport())
	cmd.AddCommand(newCmdInstallationDeletionReport())
	cmd.AddCommand(newCmdInstallationAnnotation())
	cmd.AddCommand(newCmdInstallationBackup())
	cmd.AddCommand(newCmdInstallationOperation())
	cmd.AddCommand(newCmdInstallationDNS())

	return cmd
}

func newCmdInstallationCreate() *cobra.Command {
	var flags installationCreateFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationCreateCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeInstallationCreateCmd(ctx context.Context, flags installationCreateFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	envVarMap, err := parseEnvVarInput(flags.mattermostEnv, false)
	if err != nil {
		return errors.Wrap(err, "failed to parse env var input")
	}
	priorityEnvVarMap, err := parseEnvVarInput(flags.priorityEnv, false)
	if err != nil {
		return errors.Wrap(err, "failed to parse priority env var input")
	}

	var scheduledDeletionTime int64
	if flags.scheduledDeletionTime > 0 {
		scheduledDeletionTime = model.GetMillisAtTime(time.Now().Add(flags.scheduledDeletionTime))
	}

	request := &model.CreateInstallationRequest{
		Name:                      flags.name,
		OwnerID:                   flags.ownerID,
		GroupID:                   flags.groupID,
		Version:                   flags.version,
		Image:                     flags.image,
		Size:                      flags.size,
		License:                   flags.license,
		Affinity:                  flags.affinity,
		Database:                  flags.database,
		Filestore:                 flags.filestore,
		MattermostEnv:             envVarMap,
		PriorityEnv:               priorityEnvVarMap,
		Annotations:               flags.annotations,
		GroupSelectionAnnotations: flags.groupSelectionAnnotations,
		ScheduledDeletionTime:     scheduledDeletionTime,
		PodProbeOverrides:         flags.generateProbeOverrides(),
	}

	// For CLI to be backward compatible, if only one DNS is passed we use
	// the old field.
	// TODO: properly replace with DNSNames
	if len(flags.dns) == 1 {
		request.DNS = flags.dns[0] //nolint
	} else {
		request.DNSNames = flags.dns
	}

	if model.IsSingleTenantRDS(flags.database) {
		dbConfig := model.SingleTenantDatabaseRequest{
			PrimaryInstanceType: flags.rdsPrimaryInstance,
			ReplicaInstanceType: flags.rdsReplicaInstance,
			ReplicasCount:       flags.rdsReplicasCount,
		}

		request.SingleTenantDatabaseConfig = dbConfig
	}

	if flags.database == model.InstallationDatabaseExternal {
		request.ExternalDatabaseConfig = model.ExternalDatabaseRequest{
			SecretName: flags.externalDatabaseSecretName,
		}
	}

	if flags.dryRun {
		return runDryRun(request)
	}

	installation, err := client.CreateInstallation(request)
	if err != nil {
		return errors.Wrap(err, "failed to create installation")
	}

	return printJSON(installation)
}

func newCmdInstallationUpdate() *cobra.Command {
	var flags installationUpdateFlags

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an installation's configuration.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			request, err := flags.GetPatchInstallationRequest()
			if err != nil {
				return err
			}

			if flags.dryRun {
				return runDryRun(request)
			}

			installation, err := client.UpdateInstallation(flags.installationID, request)
			if err != nil {
				return errors.Wrap(err, "failed to update installation")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.installationPatchRequestChanges.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationVolumeCreate() *cobra.Command {
	var flags installationCreateVolumeFlags

	cmd := &cobra.Command{
		Use:   "create-volume",
		Short: "Creates a new volume for an installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			request := flags.GetCreateInstallationVolumeRequest()

			if flags.dryRun {
				return runDryRun(request)
			}

			installation, err := client.CreateInstallationVolume(flags.installationID, request)
			if err != nil {
				return errors.Wrap(err, "failed to create installation volume")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationVolumeUpdate() *cobra.Command {
	var flags installationUpdateVolumeFlags

	cmd := &cobra.Command{
		Use:   "update-volume",
		Short: "Updates an existing volume for an installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			err := flags.Validate()
			if err != nil {
				return err
			}
			request := flags.GetUpdateInstallationVolumeRequest()

			if flags.dryRun {
				return runDryRun(request)
			}

			installation, err := client.UpdateInstallationVolume(flags.installationID, flags.volumeName, request)
			if err != nil {
				return errors.Wrap(err, "failed to update installation volume")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.installationUpdateVolumeChanges.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationVolumeDelete() *cobra.Command {
	var flags installationDeleteVolumeFlags

	cmd := &cobra.Command{
		Use:   "delete-volume",
		Short: "Delete an existing volume from an installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			installation, err := client.DeleteInstallationVolume(flags.installationID, flags.volumeName)
			if err != nil {
				return errors.Wrap(err, "failed to delete installation volume")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationDelete() *cobra.Command {
	var flags installationDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			if err := client.DeleteInstallation(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to delete installation")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationUpdateDeletion() *cobra.Command {
	var flags installationUpdateDeletionFlags

	cmd := &cobra.Command{
		Use:   "update-deletion",
		Short: "Updates the pending deletion parameters of an installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			request := &model.PatchInstallationDeletionRequest{}
			if flags.installationDeletionPatchRequestOptionsChanged.futureDeletionTimeChanged {
				newExpiryTimeMillis := model.GetMillisAtTime(time.Now().Add(flags.futureDeletionTime))
				request.DeletionPendingExpiry = &newExpiryTimeMillis
			}

			if flags.dryRun {
				return runDryRun(request)
			}

			installation, err := client.UpdateInstallationDeletion(flags.installationID, request)
			if err != nil {
				return errors.Wrap(err, "failed to update installation deletion parameters")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.installationDeletionPatchRequestOptionsChanged.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationCancelDeletion() *cobra.Command {
	var flags installationCancelDeletionFlags

	cmd := &cobra.Command{
		Use:   "cancel-deletion",
		Short: "Cancels the pending deletion of an installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			if err := client.CancelInstallationDeletion(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to cancel installation deletion")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationScheduleDeletion() *cobra.Command {
	var flags installationScheduledDeletionFlags

	cmd := &cobra.Command{
		Use:   "schedule-deletion",
		Short: "Schedule an installation for future deletion.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			request := &model.PatchInstallationScheduledDeletionRequest{}
			if flags.scheduledDeletionTimeChanged {
				var scheduledTimeMillis int64
				if flags.scheduledDeletionTime > 0 {
					scheduledTimeMillis = model.GetMillisAtTime(time.Now().Add(flags.scheduledDeletionTime))
				}
				request.ScheduledDeletionTime = &scheduledTimeMillis
			}

			if flags.dryRun {
				return runDryRun(request)
			}

			installation, err := client.UpdateInstallationScheduledDeletion(flags.installationID, request)
			if err != nil {
				return errors.Wrap(err, "failed to update installation scheduled deletion")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.installationScheduledDeletionRequestOptionsChanged.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationHibernate() *cobra.Command {
	var flags installationHibernateFlags

	cmd := &cobra.Command{
		Use:   "hibernate",
		Short: "Put an installation into hibernation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			installation, err := client.HibernateInstallation(flags.installationID)
			if err != nil {
				return errors.Wrap(err, "failed to put installation into hibernation")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationWakeup() *cobra.Command {
	var flags installationWakeupFlags

	cmd := &cobra.Command{
		Use:   "wake-up",
		Short: "Wake an installation from hibernation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			request, err := flags.GetPatchInstallationRequest()
			if err != nil {
				return err
			}

			installation, err := client.WakeupInstallation(flags.installationID, request)
			if err != nil {
				return errors.Wrap(err, "failed to wake up installation")
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.installationPatchRequestChanges.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationGet() *cobra.Command {
	var flags installationGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			installation, err := client.GetInstallation(flags.installationID, &model.GetInstallationRequest{
				IncludeGroupConfig:          flags.includeGroupConfig,
				IncludeGroupConfigOverrides: flags.includeGroupConfigOverrides,
			})
			if err != nil {
				return errors.Wrap(err, "failed to query installation")
			}
			if installation == nil {
				return nil
			}
			if flags.hideLicense {
				hideMattermostLicense(installation.Installation)
			}
			if flags.hideEnv {
				hideMattermostEnv(installation.Installation)
			}

			return printJSON(installation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationList() *cobra.Command {
	var flags installationListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List created installations.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationListCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.installationGetRequestChanges.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeInstallationListCmd(ctx context.Context, flags installationListFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	paging := getPaging(flags.pagingFlags)

	installations, err := client.GetInstallations(&model.GetInstallationsRequest{
		OwnerID:                     flags.owner,
		GroupID:                     flags.group,
		State:                       flags.state,
		DNS:                         flags.dns,
		IncludeGroupConfig:          flags.includeGroupConfig,
		IncludeGroupConfigOverrides: flags.includeGroupConfigOverrides,
		DeletionLocked:              flags.deletionLockedFilterValue(),
		Paging:                      paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query installations")
	}

	if flags.hideLicense {
		for _, installation := range installations {
			hideMattermostLicense(installation.Installation)
		}
	}

	if flags.hideEnv {
		for _, installation := range installations {
			hideMattermostEnv(installation.Installation)
		}
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(installations))
			for _, inst := range installations {
				data = append(data, inst)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultInstallationTableData(installations)
		}

		printTable(keys, vals)
		return nil
	}

	return printJSON(installations)
}

func defaultInstallationTableData(installations []*model.InstallationDTO) ([]string, [][]string) {
	keys := []string{"ID", "STATE", "VERSION", "DATABASE", "FILESTORE", "DNS", "API-LOCKED", "DELETION-LOCKED"}
	vals := make([][]string, 0, len(installations))
	for _, installation := range installations {
		vals = append(vals, []string{
			installation.ID,
			installation.State,
			installation.Version,
			installation.Database,
			installation.Filestore,
			dnsNames(installation.DNSRecords),
			fmt.Sprintf("%t", installation.APISecurityLock),
			fmt.Sprintf("%t", installation.DeletionLocked),
		})
	}
	return keys, vals
}

func dnsNames(dnsRecords []*model.InstallationDNS) string {
	names := model.DNSNamesFromRecords(dnsRecords)
	return strings.Join(names, ", ")
}

func newCmdInstallationGetStatuses() *cobra.Command {
	var flags installationGetStatusesFlags

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get status information for all installations.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			installationsStatus, err := client.GetInstallationsStatus()
			if err != nil {
				return errors.Wrap(err, "failed to query installation status")
			}
			if installationsStatus == nil {
				return nil
			}

			return printJSON(installationsStatus)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationShowStateReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state-report",
		Short: "Shows information regarding changing installation state.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			if err := printJSON(model.GetInstallationRequestStateReport()); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// WARNING: EXPERIMENTAL
// This command runs as a client with direct store integration.
func newCmdInstallationRecovery() *cobra.Command {
	var flags installationRecoveryFlags

	cmd := &cobra.Command{
		Use:   "recover",
		Short: "recover the basic resources of a deleted installation by recreating it.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationRecoveryCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeInstallationRecoveryCmd(flags installationRecoveryFlags) error {
	logger := logger.WithFields(logrus.Fields{
		"instance":              instanceID,
		"installation":          flags.installationID,
		"installation-database": flags.databaseID,
	})

	logger.Info("Starting installation recovery from deleted state")

	sqlStore, err := sqlStore(flags.database)
	if err != nil {
		return errors.Wrap(err, "failed to create datastore")
	}

	installation, err := sqlStore.GetInstallation(flags.installationID, true, false)
	if err != nil {
		return errors.Wrap(err, "failed to get installation")
	}
	if installation == nil {
		return errors.New("installation does not exist")
	}
	if installation.State != model.InstallationStateDeleted {
		return errors.New("installation recovery can only be performed on deleted installations")
	}

	// Name/DNS could have been claimed by a new installation, so we need to check
	// that as well.
	installations, err := sqlStore.GetInstallations(&model.InstallationFilter{
		Name:   installation.Name,
		Paging: model.AllPagesNotDeleted(),
	}, false, false)
	if err != nil {
		return errors.Wrap(err, "failed to get installations filtered by DNS")
	}
	if len(installations) != 0 {
		return errors.Errorf("the requested installation's DNS is now in use by %d installations", len(installations))
	}

	// Be extra cautious until we can test other configs.
	if installation.Database != model.InstallationDatabaseMultiTenantRDSPostgres {
		return errors.Errorf("installation database type %s is not supported", installation.Database)
	}
	if installation.Filestore != model.InstallationFilestoreBifrost && installation.Filestore != model.InstallationFilestoreMultiTenantAwsS3 {
		return errors.Errorf("installation filestore type %s is not supported", installation.Filestore)
	}

	clusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		Paging:         model.AllPagesWithDeleted(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to get cluster installations")
	}
	if len(clusterInstallations) != 1 {
		return errors.Errorf("expected to find 1 cluster installation, but found %d", len(clusterInstallations))
	}
	clusterInstallation := clusterInstallations[0]

	db, err := sqlStore.GetMultitenantDatabase(flags.databaseID)
	if err != nil {
		return errors.Wrap(err, "failed to find database")
	}
	if db == nil {
		return errors.New("failed to find multitenant database")
	}

	logger.Info("Restoring AWS resources")

	awsConfig, err := awsTools.NewAWSConfig(context.TODO())
	if err != nil {
		return errors.Wrap(err, "failed to build aws configuration")
	}
	awsClient, err := awsTools.NewAWSClientWithConfig(&awsConfig, logger)
	if err != nil {
		return errors.Wrap(err, "failed to build AWS client")
	}

	err = awsClient.SecretsManagerRestoreSecret(awsTools.RDSMultitenantSecretName(installation.ID), logger)
	if err != nil {
		return errors.Wrap(err, "failed to recover AWS database secret")
	}

	logger.Info("Updating multitenant database")

	locked, err := sqlStore.LockMultitenantDatabase(db.ID, instanceID)
	if err != nil {
		return errors.Wrap(err, "failed to lock multitenant database")
	}
	if !locked {
		return errors.New("failed to acquire lock on multitenant database")
	}
	defer func() {
		unlocked, errDefer := sqlStore.UnlockMultitenantDatabase(db.ID, instanceID, false)
		if err != nil {
			logger.WithError(errDefer).Error("Failed to unlock multitenant database")
			return
		}
		if !unlocked {
			logger.Error("Failed to release lock for multitenant database")
		}
	}()

	// Refresh the database object in case updates were made.
	db, err = sqlStore.GetMultitenantDatabase(flags.databaseID)
	if err != nil {
		return errors.Wrap(err, "failed to get database")
	}

	// Handle follow-up attempts from a partial recovery.
	// NOTE: this ignores DB weighting.
	if !db.Installations.Contains(flags.installationID) {
		db.Installations.Add(flags.installationID)
		if err = sqlStore.UpdateMultitenantDatabase(db); err != nil {
			return errors.Wrap(err, "failed to add installation ID to multitenant database")
		}
	}

	logger.Infof("Setting cluster installation %s to creation-requested", clusterInstallation.ID)

	// We shouldn't need to lock this as it is a single update and nothing
	// should have a lock since it is deleted.
	if clusterInstallation.State == model.ClusterInstallationStateDeleted {
		clusterInstallation.State = model.ClusterInstallationStateCreationRequested
		err = sqlStore.RecoverClusterInstallation(clusterInstallation)
		if err != nil {
			return errors.Wrap(err, "failed to set cluster installation to creation-requested state")
		}
	}

	logger.Info("Setting installation to creation-requested")

	installation.State = model.InstallationStateCreationRequested
	err = sqlStore.RecoverInstallation(installation)
	if err != nil {
		return errors.Wrap(err, "failed to set installation to creation-requested state")
	}

	logger.Info("Installation recovery request completed")

	return nil
}

func newCmdInstallationDeploymentReport() *cobra.Command {
	var flags installationDeploymentReportFlags

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Get a report of deployment details for a given installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationDeploymentReportCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeInstallationDeploymentReportCmd(ctx context.Context, flags installationDeploymentReportFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	installation, err := client.GetInstallation(flags.installationID, &model.GetInstallationRequest{
		IncludeGroupConfig:          true,
		IncludeGroupConfigOverrides: true,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query installation")
	}
	if installation == nil {
		return nil
	}

	output := fmt.Sprintf("Installation: %s\n", installation.ID)
	output += fmt.Sprintf(" ├ Created: %s\n", installation.CreationDateString())
	output += fmt.Sprintf(" ├ State: %s\n", installation.State)
	switch installation.State {
	case model.InstallationStateDeleted:
		output += fmt.Sprintf(" │ └ Deleted: %s\n", installation.DeletionDateString())
	case model.InstallationStateDeletionPending:
		output += fmt.Sprintf(" │ └ Deletion Pending Expiry: %s\n", installation.DeletionPendingExpiryCompleteTimeString())
	default:
		scheduledDeletion := installation.ScheculedDeletionCompleteTimeString()
		if scheduledDeletion != "n/a" {
			output += fmt.Sprintf(" │ └ Scheduled Deletion: %s\n", scheduledDeletion)
		}
	}
	output += fmt.Sprintf(" ├ DNS: %s (primary)\n", installation.DNS) //nolint
	if len(installation.DNSRecords) > 1 {
		var alternateDNS []string
		for _, record := range installation.DNSRecords {
			if !record.IsPrimary {
				alternateDNS = append(alternateDNS, record.DomainName)
			}
		}
		output += fmt.Sprintf(" │ └ Alternate Domains: %s\n", strings.Join(alternateDNS, ", "))
	}
	output += fmt.Sprintf(" ├ Version: %s:%s\n", installation.Image, installation.Version)
	output += fmt.Sprintf(" ├ Size: %s\n", installation.Size)
	output += fmt.Sprintf(" ├ Affinity: %s\n", installation.Affinity)
	output += fmt.Sprintf(" ├ Environment Variables: %d\n", len(installation.MattermostEnv))
	if len(installation.PriorityEnv) != 0 {
		var envKeys []string
		for key := range installation.PriorityEnv {
			envKeys = append(envKeys, key)
		}
		output += fmt.Sprintf(" ├ Priority Environment Variables: %d\n", len(installation.PriorityEnv))
		output += fmt.Sprintf(" │ └ Env Keys Set: %s\n", strings.Join(envKeys, ", "))
	}
	output += " ├ Locks:\n"
	output += fmt.Sprintf(" │ ├ API: %v\n", installation.APISecurityLock)
	output += fmt.Sprintf(" │ ├ Deletion: %v\n", installation.DeletionLocked)
	output += fmt.Sprintf(" │ └ Provisioner: %v\n", installation.LockAcquiredAt != 0)
	output += fmt.Sprintf(" ├ Database Type: %s\n", installation.Database)
	if model.IsMultiTenantRDS(installation.Database) {
		var databases []*model.MultitenantDatabase
		databases, err = client.GetMultitenantDatabases(&model.GetMultitenantDatabasesRequest{
			Paging: model.AllPagesWithDeleted(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to query installation database")
		}
		for _, database := range databases {
			if database.Installations.Contains(installation.ID) {
				output += fmt.Sprintf(" │ ├ Database: %s\n", database.ID)
				output += fmt.Sprintf(" │ ├ State: %s\n", database.State)
				output += fmt.Sprintf(" │ ├ VPC: %s\n", database.VpcID)
				output += fmt.Sprintf(" │ ├ Database Writer Endpoint: %s\n", database.WriterEndpoint)
				output += fmt.Sprintf(" │ └ Installations: %d\n", len(database.Installations))
			}
		}
		if installation.Database == model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer {
			var schemas []*model.DatabaseSchema
			schemas, err = client.GetDatabaseSchemas(&model.GetDatabaseSchemaRequest{
				InstallationID: flags.installationID,
				Paging:         model.AllPagesWithDeleted(),
			})
			if err != nil {
				return errors.Wrap(err, "failed to query installation database schema")
			}
			for _, schema := range schemas {
				var logicalDatabase *model.LogicalDatabase
				logicalDatabase, err = client.GetLogicalDatabase(schema.LogicalDatabaseID)
				if err != nil {
					return errors.Wrap(err, "failed to query installation logical database")
				}
				var schemasInLogicalDatabase []*model.DatabaseSchema
				schemasInLogicalDatabase, err = client.GetDatabaseSchemas(&model.GetDatabaseSchemaRequest{
					LogicalDatabaseID: logicalDatabase.ID,
					Paging:            model.AllPagesNotDeleted(),
				})
				if err != nil {
					return errors.Wrap(err, "failed to query schemas in logical database")
				}
				output += fmt.Sprintf(" │   ├ Logical Database: %s\n", logicalDatabase.ID)
				output += fmt.Sprintf(" │   ├ Name: %s\n", logicalDatabase.Name)
				output += fmt.Sprintf(" │   └ Schemas: %d\n", len(schemasInLogicalDatabase))
				output += fmt.Sprintf(" │     ├ Database Schema: %s\n", schema.ID)
				output += fmt.Sprintf(" │     └ Name: %s\n", schema.Name)
			}
		}
	} else {
		switch installation.Database {
		case model.InstallationDatabaseSingleTenantRDSMySQL,
			model.InstallationDatabaseSingleTenantRDSPostgres:
			output += fmt.Sprintf(" │ ├ Primary Instance Type: %s\n", installation.SingleTenantDatabaseConfig.PrimaryInstanceType)
			output += fmt.Sprintf(" │ ├ Replica Instance Type: %s\n", installation.SingleTenantDatabaseConfig.ReplicaInstanceType)
			output += fmt.Sprintf(" │ └ Replica Count: %d\n", installation.SingleTenantDatabaseConfig.ReplicasCount)
		case model.InstallationDatabaseExternal:
			output += fmt.Sprintf(" │ └ Database Secret Name: %s\n", installation.ExternalDatabaseConfig.SecretName)
		}
	}
	output += fmt.Sprintf(" ├ Filestore Type: %s\n", installation.Filestore)

	if installation.GroupID != nil && len(*installation.GroupID) != 0 {
		var group *model.GroupDTO
		group, err = client.GetGroup(*installation.GroupID)
		if err != nil {
			return errors.Wrap(err, "failed to query installation group")
		}
		output += fmt.Sprintf(" ├ Group: %s\n", group.ID)
		output += fmt.Sprintf(" │ ├ Name: %s\n", group.Name)
		output += fmt.Sprintf(" │ └ Description: %s\n", group.Description)
	}

	clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
		InstallationID: installation.ID,
		Paging:         model.AllPagesWithDeleted(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to query cluster installations")
	}
	for _, clusterInstallation := range clusterInstallations {
		output += fmt.Sprintf(" └ Cluster Installation: %s\n", clusterInstallation.ID)

		cluster, err := client.GetCluster(clusterInstallation.ClusterID)
		if err != nil {
			return errors.Wrap(err, "failed to query cluster")
		}

		var provisionerMetadata model.ProvisionerMetadata
		var masterCount int64
		if cluster.Provisioner == model.ProvisionerKops && cluster.ProvisionerMetadataKops != nil {
			provisionerMetadata = cluster.ProvisionerMetadataKops.GetCommonMetadata()
			masterCount = cluster.ProvisionerMetadataKops.MasterCount
		} else if cluster.Provisioner == model.ProvisionerEKS && cluster.ProvisionerMetadataEKS != nil {
			provisionerMetadata = cluster.ProvisionerMetadataEKS.GetCommonMetadata()
			masterCount = 1
		}

		output += fmt.Sprintf("   ├ State: %s\n", clusterInstallation.State)
		output += fmt.Sprintf("   └ Cluster: %s\n", cluster.ID)
		output += fmt.Sprintf("     ├ State: %s\n", cluster.State)
		output += fmt.Sprintf("     ├ VPC: %s\n", provisionerMetadata.VPC)
		output += fmt.Sprintf("     ├ Nodes: Masters %d, Workers %d\n", masterCount, provisionerMetadata.NodeMaxCount)
		output += fmt.Sprintf("     └ Version: %s\n", provisionerMetadata.Version)
	}

	if flags.eventCount > 0 {
		output += "\nRecent Events:\n"

		req := model.ListStateChangeEventsRequest{
			Paging: model.Paging{
				Page:           0,
				PerPage:        flags.eventCount,
				IncludeDeleted: false,
			},
			ResourceType: model.ResourceType("installation"),
			ResourceID:   flags.installationID,
		}

		events, err := client.ListStateChangeEvents(&req)
		if err != nil {
			return err
		}
		for _, event := range events {
			output += fmt.Sprintf("%s - %s > %s\n", model.DateTimeStringFromMillis(event.Event.Timestamp), event.StateChange.OldState, event.StateChange.NewState)
		}
	}

	fmt.Println(output)
	return nil
}

func newCmdInstallationDeletionReport() *cobra.Command {
	var flags installationDeletionReportFlags

	cmd := &cobra.Command{
		Use:   "deletion-report",
		Short: "Get a report of installation deletion pending times.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationDeletionReportCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeInstallationDeletionReportCmd(ctx context.Context, flags installationDeletionReportFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	installations, err := client.GetInstallations(&model.GetInstallationsRequest{
		State:  model.InstallationStateDeletionPending,
		Paging: model.AllPagesNotDeleted(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to query installations")
	}

	// Prepare the time cutoffs for the report.
	now := time.Now()
	var report model.DeletionPendingReport
	for i := 1; i <= flags.days; i++ {
		report.NewCutoff(fmt.Sprintf("%d day(s)", i), now.Add(time.Duration(i)*24*time.Hour))
	}

	for _, installation := range installations {
		report.Count(installation.DeletionPendingExpiry)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"TIME TO DELETION", "COUNT"})
	for _, cutoff := range report.Cutoffs {
		table.Append([]string{cutoff.Name, toStr(cutoff.Count)})
	}
	table.Append([]string{"Sometime later", toStr(report.Overflow)})

	table.Render()
	return nil
}

func hideMattermostLicense(installation *model.Installation) {
	if len(installation.License) != 0 {
		installation.License = hiddenLicense
	}
}

func hideMattermostEnv(installation *model.Installation) {
	if installation.MattermostEnv != nil {
		installation.MattermostEnv = model.EnvVarMap{
			fmt.Sprintf("hidden (%d env vars) (--hide-env=true)", len(installation.MattermostEnv)): model.EnvVar{},
		}
	}
}
