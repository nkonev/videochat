package db

import (
	"database/sql"
	rice "github.com/GeertJohan/go.rice"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	. "github.com/nkonev/videochat/logger"
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
	db, err := sql.Open(postgresString, conninfo)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(maxLifetime)
	db.SetMaxIdleConns(maxIdle)
	db.SetMaxOpenConns(maxOpen)
	return &DB{db}, nil
}

// Begin starts an returns a new transaction.
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
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
