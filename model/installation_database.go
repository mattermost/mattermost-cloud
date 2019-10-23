package model

const (
	// InstallationDatabaseMysqlOperator is a database hosted in kubernetes via the operator.
	InstallationDatabaseMysqlOperator = "mysql-operator"
	// InstallationDatabaseAwsRDS is a database hosted via Amazon RDS.
	InstallationDatabaseAwsRDS = "aws-rds"
)

// IsSupportedDatabase returns true if the given database string is supported.
func IsSupportedDatabase(database string) bool {
	return database == InstallationDatabaseMysqlOperator || database == InstallationDatabaseAwsRDS
}
