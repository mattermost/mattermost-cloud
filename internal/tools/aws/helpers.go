package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

// DefaultAWSRegions returns the default AWS regions used by the provisioner.
var DefaultAWSRegions []*string

func init() {
	DefaultAWSRegions = []*string{
		aws.String("us-east-1a"),
		aws.String("us-east-1b"),
		aws.String("us-east-1c"),
	}
}

// CloudID returns the standard ID used for AWS resource names. This ID is used
// to correlate installations to AWS resources.
func CloudID(id string) string {
	return cloudIDPrefix + id
}

// RDSSnapshotTagValue returns the value for tagging a RDS snapshot.
func RDSSnapshotTagValue(cloudID string) string {
	return fmt.Sprintf("rds-snapshot-%s", cloudID)
}

// IAMSecretName returns the IAM Access Key secret name for a given Cloud ID.
func IAMSecretName(cloudID string) string {
	return cloudID + iamSuffix
}

// RDSSecretName returns the RDS secret name for a given Cloud ID.
func RDSSecretName(cloudID string) string {
	return cloudID + rdsSuffix
}

func trimTagPrefix(tag string) string {
	return strings.TrimLeft(tag, "tag:")
}

const passwordBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func newRandomPassword(length int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = passwordBytes[rand.Intn(len(passwordBytes))]
	}

	return string(b)
}

// DBSubnetGroupName formats the subnet group name used for RDS databases.
func DBSubnetGroupName(vpcID string) string {
	return fmt.Sprintf("mattermost-provisioner-db-%s", vpcID)
}

// RDSMasterInstanceID formats the id used for RDS database instances.
func RDSMasterInstanceID(installationID string) string {
	return fmt.Sprintf("%s-master", CloudID(installationID))
}

// RDSMigrationClusterID formats the id used for migrating RDS database cluster.
func RDSMigrationClusterID(installationID string) string {
	return fmt.Sprintf("%s-migration", CloudID(installationID))
}

// RDSMigrationMasterInstanceID formats the id used for migrating RDS database instances.
func RDSMigrationMasterInstanceID(installationID string) string {
	return fmt.Sprintf("%s-master", RDSMigrationClusterID(installationID))
}

// IsErrorCode asserts that an AWS error has a certain code.
func IsErrorCode(err error, code string) bool {
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if ok {
			return awsErr.Code() == code
		}
	}
	return false
}

// DatabaseStatus ..
type DatabaseStatus struct {
	TimeBehindMaster     int    `gorm:"column:Time_Behind_Master"`
	ReadMasterLogPos     int    `gorm:"column:Read_Master_Log_Pos"`
	RelayLogPos          int    `gorm:"column:Relay_Log_Pos"`
	RelayLogFile         string `gorm:"column:Relay_Log_File"`
	SlaveSQLRunningState string `gorm:"column:Slave_SQL_Running_State"`
	SlaveIORunning       string `gorm:"column:Slave_IO_Running"`
}

// SQLClient ...
type SQLClient interface {
	Connect(connString string) error
	Status() *DatabaseStatus
	SetBinlogRetention(hours uint64) error
	CreateReplicationUser(username, secret string) error
	Close() error
}

// MySQLClient ..
type MySQLClient struct {
	db *gorm.DB
}

// Connect fmt.Sprintf("%s:%s@tcp(%s:3306)/?charset=utf8&parseTime=True", d.user, d.pass, d.address)
func (d *MySQLClient) Connect(connString string) error {
	db, err := gorm.Open("mysql", connString)
	if err != nil {
		return err
	}
	d.db = db

	return nil
}

// Status ...
func (d *MySQLClient) Status() *DatabaseStatus {
	status := DatabaseStatus{}
	d.db.Raw("SHOW SLAVE STATUS").Scan(&status)
	return &status
}

// SetBinlogRetention ...
func (d *MySQLClient) SetBinlogRetention(hours uint64) error {
	_, err := d.db.DB().Query(fmt.Sprintf("CALL mysql.rds_set_configuration('binlog retention hours', %v)", hours))
	if err != nil {
		return err
	}

	return nil
}

// CreateReplicationUser ...
func (d *MySQLClient) CreateReplicationUser(username, secret string) error {
	_, err := d.db.DB().Query("FLUSH PRIVILEGES")
	if err != nil {
		return err
	}

	_, err = d.db.DB().Query("CREATE USER '" + username + "'@'%' IDENTIFIED BY '" + secret + "'")
	if err != nil {
		return err
	}

	_, err = d.db.DB().Query("GRANT REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO '" + username + "'@'%'")
	if err != nil {
		return err
	}

	return nil
}

// Close ...
func (d *MySQLClient) Close() error {
	return d.db.Close()
}
