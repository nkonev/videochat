package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	pgxSlog "github.com/mcosta74/pgx-slog"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.uber.org/fx"
)

type DB struct {
	pool *pgxpool.Pool
	lgr  *logger.LoggerWrapper
	*sql.DB
}

type Tx struct {
	*sql.Tx
	lgr *logger.LoggerWrapper
}

// Transactional and non- operations
type CommonOperations interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
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

// Begin starts and returns a new transaction.
func (db *DB) Begin(ctx context.Context, lgr *logger.LoggerWrapper) (*Tx, error) {
	if tx, err := db.DB.BeginTx(ctx, nil); err != nil {
		return nil, fmt.Errorf("error during interacting with db: %w", err)
	} else {
		return &Tx{tx, lgr}, nil
	}
}

func (tx *Tx) SafeRollback() {
	if err0 := tx.Rollback(); err0 != nil {
		tx.lgr.Error("Error during rollback tx ", logger.AttributeError, err0)
	}
}

type TraceLogWrapper struct {
	delegate tracelog.TraceLog
}

// https://github.com/jackc/pgx/pull/2482
func (tl *TraceLogWrapper) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	logArgs := make([]any, 0, len(data.Args))

	for i, a := range data.Args {
		if i == 0 {
			switch a.(type) {
			case pgx.QueryResultFormats:
				continue
			case pgx.QueryResultFormatsByOID:
				continue
			case pgx.QueryExecMode:
				continue
			case pgx.QueryRewriter:
				continue
			}
		}
		logArgs = append(logArgs, a)
	}

	data.Args = logArgs

	return tl.delegate.TraceQueryStart(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	tl.delegate.TraceQueryEnd(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceBatchStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	return tl.delegate.TraceBatchStart(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	tl.delegate.TraceBatchQuery(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	tl.delegate.TraceBatchEnd(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceCopyFromStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	return tl.delegate.TraceCopyFromStart(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData) {
	tl.delegate.TraceCopyFromEnd(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	//return tl.delegate.TraceConnectStart(ctx, data)
	return ctx
}

func (tl *TraceLogWrapper) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	//tl.delegate.TraceConnectEnd(ctx, data)
}

func (tl *TraceLogWrapper) TracePrepareStart(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	// return tl.delegate.TracePrepareStart(ctx, conn, data)
	return ctx
}

func (tl *TraceLogWrapper) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
	// tl.delegate.TracePrepareEnd(ctx, conn, data)
}

func (tl *TraceLogWrapper) TraceAcquireStart(ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireStartData) context.Context {
	return tl.delegate.TraceAcquireStart(ctx, pool, data)
}

func (tl *TraceLogWrapper) TraceAcquireEnd(ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	tl.delegate.TraceAcquireEnd(ctx, pool, data)
}

func (tl *TraceLogWrapper) TraceRelease(pool *pgxpool.Pool, data pgxpool.TraceReleaseData) {
	tl.delegate.TraceRelease(pool, data)
}

func ConfigureDatabase(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	tp *sdktrace.TracerProvider,
	lc fx.Lifecycle,
) (*DB, error) {
	lgr.Info("Creating database pool")

	config, err := pgxpool.ParseConfig(cfg.PostgreSQL.Url)
	if err != nil {
		return nil, err
	}

	config.MaxConns = int32(cfg.PostgreSQL.MaxOpenConnections)
	config.MinIdleConns = int32(cfg.PostgreSQL.MaxIdleConnections)
	config.MaxConnLifetime = cfg.PostgreSQL.MaxLifetime

	// https://github.com/mcosta74/pgx-slog
	adapterLogger := pgxSlog.NewLogger(lgr.Logger)

	tracers := []pgx.QueryTracer{
		otelpgx.NewTracer(otelpgx.WithTracerProvider(tp)),
	}

	if cfg.PostgreSQL.Dump {
		ll, err := tracelog.LogLevelFromString(cfg.PostgreSQL.LogLevel)
		if err != nil {
			return nil, err
		}

		tracers = append(tracers, &TraceLogWrapper{tracelog.TraceLog{
			Logger:   adapterLogger,
			LogLevel: ll,
			Config: &tracelog.TraceLogConfig{
				TimeKey: "duration",
			},
		}})
	}

	// https://github.com/jackc/pgx/discussions/1677#discussioncomment-12253699
	m := multitracer.New(tracers...)

	config.ConnConfig.Tracer = m

	ctx := context.Background()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	stdDb := stdlib.OpenDBFromPool(pool)

	if lc != nil {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				lgr.Info("Stopping database pool")
				stdDb.Close()
				pool.Close()
				return nil
			},
		})
	}

	return &DB{pool, lgr, stdDb}, nil
}

//go:embed migrations
var embeddedMigrationFiles embed.FS

func (db *DB) Migrate(mc config.MigrationConfig) error {
	db.lgr.Info("Starting migration")
	staticDir := http.FS(embeddedMigrationFiles)
	src, err := httpfs.New(staticDir, "migrations")
	if err != nil {
		return err
	}

	// here we acquire a dedicated sql db in order to properly close it to prevent hanging on the app shutdown
	stdDb := stdlib.OpenDBFromPool(db.pool)
	defer stdDb.Close()

	pgInstance, err := pgxMigrate.WithInstance(stdDb, &pgxMigrate.Config{
		MigrationsTable:  mc.MigrationTable,
		StatementTimeout: mc.StatementDuration,
	})
	if err != nil {
		return err
	}
	defer pgInstance.Close()

	m, err := migrate.NewWithInstance("httpfs", src, "", pgInstance)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	db.lgr.Info("Migration successfully completed")
	return nil
}

func (db *DB) Reset(mc config.MigrationConfig, hard bool) error {

	additional := ""
	if hard { // whether to preserve techical info (need to fast-worward, need to skip old db migration and so on)
		additional = `
		drop table if exists technical;
		`
	}

	_, err := db.Exec(fmt.Sprintf(`
	drop EXTENSION if exists pg_trgm;
	drop sequence if exists chat_id_sequence;
	
	drop table if exists chat_common;
	drop table if exists chat_participant;
	drop table if exists message_published;
	drop table if exists message_pinned;
	drop table if exists message_reaction;
	drop table if exists message;
	drop table if exists chat_user_view;
	drop table if exists has_unread_messages;
	
	%s

	drop table if exists blog;

	drop FUNCTION if exists strip_tags;
	drop FUNCTION if exists cyrillic_transliterate;

	drop table if exists %s;

	-- transaction_utils_test.go
	drop table if exists t1;
	drop table if exists t2;
	drop table if exists tr1;
	drop table if exists tr2;

`, additional, mc.MigrationTable))
	db.lgr.Info("Recreating database", logger.AttributeError, err)
	return err
}

func RunMigrations(db *DB, cfg *config.AppConfig) error {
	return db.Migrate(cfg.PostgreSQL.Migration)
}

func RunResetDatabaseSoft(db *DB, cfg *config.AppConfig) error {
	return db.Reset(cfg.PostgreSQL.Migration, false)
}

func RunResetDatabaseHard(db *DB, cfg *config.AppConfig) error {
	return db.Reset(cfg.PostgreSQL.Migration, true)
}
