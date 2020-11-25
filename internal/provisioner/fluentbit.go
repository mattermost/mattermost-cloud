// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type fluentbit struct {
	provisioner    *KopsProvisioner
	awsClient      aws.AWS
	kops           *kops.Cmd
	logger         log.FieldLogger
	desiredVersion model.UtilityVersion
	actualVersion  model.UtilityVersion
}

func newFluentbitHandle(version model.UtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*fluentbit, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Fluentbit handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Fluentbit if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Fluentbit if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Fluentbit if the Kops command provided is nil")
	}

	return &fluentbit{
		provisioner:    provisioner,
		awsClient:      awsClient,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.FluentbitCanonicalName),
		desiredVersion: version,
	}, nil
}

func (f *fluentbit) Destroy() error {
	return nil
}

func (f *fluentbit) Migrate() error {
	return nil
}

func (f *fluentbit) CreateOrUpgrade() error {
	logger := f.logger.WithField("fluentbit-action", "upgrade")
	h := f.NewHelmDeployment(logger)

	err := h.Update()
	if err != nil {
		return err
	}

	err = f.updateVersion(h)
	return err
}

func (f *fluentbit) DesiredVersion() model.UtilityVersion {
	return f.desiredVersion
}

func (f *fluentbit) ActualVersion() model.UtilityVersion {
	if f.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(f.actualVersion.Version(), "fluent-bit-"),
		ValuesPath: f.actualVersion.Values(),
	}
}

func (f *fluentbit) Name() string {
	return model.FluentbitCanonicalName
}

func (f *fluentbit) NewHelmDeployment(logger log.FieldLogger) *helmDeployment {
	privateDomainName, err := f.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		logger.WithError(err).Error("unable to lookup private zone name")
	}

	var auditLogsConf string
	zoneID, err := f.awsClient.GetPrivateZoneIDForDefaultTag(logger)
	if err != nil {
		logger.WithError(err).Error("unable to get Private Zone ID with the default tag, skipping setup...")
	} else {
		tag, err := f.awsClient.GetTagByKeyAndZoneID(aws.DefaultAuditLogsCoreSecurityTagKey, zoneID, logger)
		if err != nil {
			logger.WithError(err).Errorf("unable to find %s", aws.DefaultAuditLogsCoreSecurityTagKey)
		}
		if tag == nil {
			logger.Infof("%s is missing, skipping setup...", aws.DefaultAuditLogsCoreSecurityTagKey)
			tag = &aws.Tag{}
		}

		hostPort := strings.Split(tag.Value, ":")
		if len(hostPort) == 2 {
			auditLogsConf = fmt.Sprintf(`[OUTPUT]
	Name  forward
	Match  *
	Host  %s
	Port  %s
	tls  On
	tls.verify  Off`, hostPort[0], hostPort[1])
		} else {
			logger.Info("AuditLogsCoreSecurity tag is missing from R53 hosted zone, " +
				"fluent-bit will be configured without forwarding to audit logs to Security")
		}
	}

	elasticSearchDNS := fmt.Sprintf("elasticsearch.%s", privateDomainName)
	return &helmDeployment{
		chartDeploymentName: "fluent-bit",
		chartName:           "stable/fluent-bit",
		namespace:           "fluent-bit",
		setArgument: fmt.Sprintf(`backend.es.host=%s,rawConfig=
@INCLUDE fluent-bit-service.conf
@INCLUDE fluent-bit-input.conf
@INCLUDE fluent-bit-filter.conf
@INCLUDE fluent-bit-output.conf
%s
`, elasticSearchDNS, auditLogsConf),
		kopsProvisioner: f.provisioner,
		kops:            f.kops,
		logger:          f.logger,
		desiredVersion:  f.desiredVersion,
	}
}

func (f *fluentbit) ValuesPath() string {
	if f.desiredVersion == nil {
		return ""
	}
	return f.desiredVersion.Values()
}

func (f *fluentbit) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	f.actualVersion = actualVersion
	return nil
}
