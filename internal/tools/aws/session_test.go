package aws

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func (a *AWSTestSuite) TestDefaultOpts() {
	opts := DefaultSessionConfig()

	a.Assert().NotNil(opts)
	a.Assert().Equal(DefaultAWSClientRetries, *opts.Opts.Config.MaxRetries)
	a.Assert().Equal(DefaultAWSRegion, *opts.Opts.Config.Region)
	a.Assert().Equal(opts, DefaultSessionConfig())
}

func (a *AWSTestSuite) TestCreateSession() {
	sess, err := DefaultSessionConfig().CreateSession(a.Mocks.LOG.Logger)
	a.Assert().NoError(err)
	a.Assert().NotNil(sess)
}

func (a *AWSTestSuite) TestLogHandler() {
	sess, err := DefaultSessionConfig().CreateSession(a.Mocks.LOG.Logger)
	a.Assert().NoError(err)
	a.Assert().NotNil(sess)

	a.Mocks.LOG.Logger.On("Debugf", "%s: %s%s\n%s", "POST", "rds.us-east-1.amazonaws.com", "",
		mock.AnythingOfType("*rds.CreateDBClusterSnapshotInput")).Return(logrus.NewEntry(&logrus.Logger{}))

	err = NewAWSClient(sess).CreateDatabaseSnapshot(a.InstallationA.ID)
	a.Assert().Error(err)
}
