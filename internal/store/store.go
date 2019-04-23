package store

import (
	"database/sql"
	"net/url"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	// enable the pq driver
	_ "github.com/lib/pq"
	// enable the sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// SQLStore abstracts access to the database.
type SQLStore struct {
	db     *sqlx.DB
	logger logrus.FieldLogger
}

// New constructs a new instance of SQLStore.
func New(dsn string, logger logrus.FieldLogger) (*SQLStore, error) {
	url, err := url.Parse(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse dsn as an url")
	}

	var db *sqlx.DB

	switch strings.ToLower(url.Scheme) {
	case "sqlite", "sqlite3":
		db, err = sqlx.Connect("sqlite3", url.Host)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to sqlite database")
		}

		// Override the default mapper to use the field names "as-is"
		db.MapperFunc(func(s string) string { return s })

	case "postgres", "postgresql":
		url.Scheme = "postgres"
		db, err = sqlx.Connect("postgres", url.String())
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to postgres database")
		}

		// Leave the default mapper as strings.ToLower.

	default:
		return nil, errors.Errorf("unsupported dsn scheme %s", url.Scheme)
	}

	return &SQLStore{
		db,
		logger,
	}, nil
}

// queryer is an interface describing a resource that can query.
//
// It exactly matches sqlx.Queryer, existing simply to constrain sqlx usage to this file.
type queryer interface {
	sqlx.Queryer
}

// get queries for a single row, writing the result into dest.
//
// Use this to simplify querying for a single row or column. Dest may be a pointer to a simple
// type, or a struct with fields to be populated from the returned columns.
func (sqlStore *SQLStore) get(q sqlx.Queryer, dest interface{}, query string, args ...interface{}) error {
	query = sqlStore.db.Rebind(query)

	return sqlx.Get(q, dest, query, args...)
}

// builder is an interface describing a resource that can construct SQL and arguments.
//
// It exists to allow consuming any squirrel.*Builder type.
type builder interface {
	ToSql() (string, []interface{}, error)
}

// get queries for a single row, building the sql, and writing the result into dest.
//
// Use this to simplify querying for a single row or column. Dest may be a pointer to a simple
// type, or a struct with fields to be populated from the returned columns.
func (sqlStore *SQLStore) getBuilder(q sqlx.Queryer, dest interface{}, b builder) error {
	sql, args, err := b.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build sql")
	}

	sql = sqlStore.db.Rebind(sql)

	err = sqlx.Get(q, dest, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

// selectBuilder queries for one or more rows, building the sql, and writing the result into dest.
//
// Use this to simplify querying for multiple rows (and possibly columns). Dest may be a slice of
// a simple, or a slice of a struct with fields to be populated from the returned columns.
func (sqlStore *SQLStore) selectBuilder(q sqlx.Queryer, dest interface{}, b builder) error {
	sql, args, err := b.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build sql")
	}

	sql = sqlStore.db.Rebind(sql)

	err = sqlx.Select(q, dest, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

// execer is an interface describing a resource that can execute write queries.
//
// It allows the use of *sqlx.Db and *sqlx.Tx.
type execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// exec executes the given query using positional arguments, automatically rebinding for the db.
func (sqlStore *SQLStore) exec(e execer, sql string, args ...interface{}) (sql.Result, error) {
	sql = sqlStore.db.Rebind(sql)
	return e.Exec(sql, args...)
}

// exec executes the given query, building the necessary sql.
func (sqlStore *SQLStore) execBuilder(e execer, b builder) (sql.Result, error) {
	sql, args, err := b.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build sql")
	}

	return sqlStore.exec(e, sql, args...)
}
