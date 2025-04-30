package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"net/http"
	"nkonev.name/video/logger"
	"time"
)

const prodMigrationsTable = "go_migrate"
const testMigrationsTable = "go_migrate_test"

// https://medium.com/@benbjohnson/structuring-applications-in-go-3b04be4ff091
type DB struct {
	*sql.DB
	lgr *logger.Logger
}

type Tx struct {
	*sql.Tx
	lgr *logger.Logger
}

func (dbR *DB) logger() *logger.Logger {
	return dbR.lgr
}

func (tx *Tx) logger() *logger.Logger {
	return tx.lgr
}

type MigrationsConfig struct {
	AppendTestData bool
}

// enumerates common tx and non-tx operations
type CommonOperations interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	logger() *logger.Logger
}

func (dbR *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return dbR.DB.QueryContext(ctx, query, args...)
}

func (txR *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return txR.Tx.QueryContext(ctx, query, args...)
}

func (dbR *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return dbR.DB.QueryRowContext(ctx, query, args...)
}

func (txR *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return txR.Tx.QueryRowContext(ctx, query, args...)
}

func (dbR *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return dbR.DB.ExecContext(ctx, query, args...)
}

func (txR *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return txR.Tx.ExecContext(ctx, query, args...)
}

const postgresDriverString = "pgx"

// Open returns a DB reference for a data source.
func Open(lgr *logger.Logger, conninfo string, maxOpen int, maxIdle int, maxLifetime time.Duration) (*DB, error) {
	if db, err := sql.Open(postgresDriverString, conninfo); err != nil {
		return nil, err
	} else {
		db.SetConnMaxLifetime(maxLifetime)
		db.SetMaxIdleConns(maxIdle)
		db.SetMaxOpenConns(maxOpen)
		return &DB{db, lgr}, nil
	}
}

// Begin starts an returns a new transaction.
func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	if tx, err := db.DB.BeginTx(ctx, nil); err != nil {
		return nil, err
	} else {
		return &Tx{tx, db.lgr}, nil
	}
}

func (tx *Tx) SafeRollback() {
	if err0 := tx.Rollback(); err0 != nil {
		tx.lgr.Errorf("Error during rollback tx %v", err0)
	}
}

//go:embed migrations
var embeddedFiles embed.FS

func migrateInternal(lgr *logger.Logger, db *sql.DB, path, migrationTable string) {
	staticDir := http.FS(embeddedFiles)
	src, err := httpfs.New(staticDir, "migrations"+path)
	if err != nil {
		lgr.Fatal(err)
	}

	d, err := time.ParseDuration("15m")
	if err != nil {
		lgr.Fatal(err)
	}

	pgInstance, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable:  migrationTable,
		StatementTimeout: d,
	})
	if err != nil {
		lgr.Fatal(err)
	}

	m, err := migrate.NewWithInstance("httpfs", src, "", pgInstance)
	if err != nil {
		lgr.Fatal(err)
	}
	//defer m.Close()
	if err := m.Up(); err != nil && err.Error() != "no change" {
		lgr.Fatal(err)
	}
}

func (db *DB) Migrate(lgr *logger.Logger, migrationsConfig *MigrationsConfig) {
	lgr.Infof("Starting prod migration")
	migrateInternal(lgr, db.DB, "/prod", prodMigrationsTable)
	lgr.Infof("Migration successful prod completed")

	if migrationsConfig.AppendTestData {
		lgr.Infof("Starting test migration")
		migrateInternal(lgr, db.DB, "/test", testMigrationsTable)
		lgr.Infof("Migration successful test completed")
	}
}

func ConfigureDb(lgr *logger.Logger, lc fx.Lifecycle) (*DB, error) {
	dbConnectionString := viper.GetString("postgresql.url")
	maxOpen := viper.GetInt("postgresql.maxOpenConnections")
	maxIdle := viper.GetInt("postgresql.maxIdleConnections")
	maxLifeTime := viper.GetDuration("postgresql.maxLifetime")
	dbInstance, err := Open(lgr, dbConnectionString, maxOpen, maxIdle, maxLifeTime)

	if lc != nil {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				lgr.Infof("Stopping db connection")
				return dbInstance.Close()
			},
		})
	}

	return dbInstance, err
}

func (db *DB) RecreateDb() {
	_, err := db.Exec(fmt.Sprintf(`
	drop table if exists user_call_state;
	drop table if exists %s;
	drop table if exists %s;
	
	drop function if exists utc_now();
`, prodMigrationsTable, testMigrationsTable))
	db.lgr.Warn("Recreating database")
	if err != nil {
		db.lgr.Panicf("Error during dropping db: %v", err)
	}
}
