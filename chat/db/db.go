package db

import (
	"database/sql"
	rice "github.com/GeertJohan/go.rice"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	. "nkonev.name/chat/logger"
	"time"
)

// https://medium.com/@benbjohnson/structuring-applications-in-go-3b04be4ff091
type DB struct {
	*sql.DB
}

type Tx struct {
	*sql.Tx
}

const postgresString = "postgres"

// Open returns a DB reference for a data source.
func Open(conninfo string, maxOpen int, maxIdle int, maxLifetime time.Duration) (*DB, error) {
	if db, err := sql.Open(postgresString, conninfo); err != nil {
		return nil, err
	} else {
		db.SetConnMaxLifetime(maxLifetime)
		db.SetMaxIdleConns(maxIdle)
		db.SetMaxOpenConns(maxOpen)
		return &DB{db}, nil
	}
}

// Begin starts an returns a new transaction.
func (db *DB) Begin() (*Tx, error) {
	if tx, err := db.DB.Begin(); err != nil {
		return nil, err
	} else {
		return &Tx{tx}, nil
	}
}

func (tx *Tx) SafeRollback() {
	if err0 := tx.Rollback(); err0 != nil {
		Logger.Errorf("Error during rollback tx %v", err0)
	}
}

func migrateInternal(db *sql.DB) {
	const migrations = "migrations"
	box := rice.MustFindBox(migrations).HTTPBox()
	src, err := httpfs.New(box, ".")
	if err != nil {
		Logger.Fatal(err)
	}

	d, err := time.ParseDuration("15m")
	if err != nil {
		Logger.Fatal(err)
	}

	pgInstance, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable:  "migrations",
		StatementTimeout: d,
	})
	if err != nil {
		Logger.Fatal(err)
	}

	m, err := migrate.NewWithInstance("httpfs", src, postgresString, pgInstance)
	if err != nil {
		Logger.Fatal(err)
	}
	//defer m.Close()
	if err := m.Up(); err != nil && err.Error() != "no change" {
		Logger.Fatal(err)
	}
}

func (db *DB) Migrate() {
	migrateInternal(db.DB)
}
