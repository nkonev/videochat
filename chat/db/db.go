package db

import (
	"context"
	"database/sql"
	dbP "database/sql"
	"embed"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"net/http"
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

type MigrationsConfig struct {
	AppendTestData bool
}

type UserAdminDbDTO struct {
	UserId int64
	Admin  bool
}

// enumerates common tx and non-tx operations
type CommonOperations interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*dbP.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *dbP.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	GetParticipantIds(ctx context.Context, chatId int64, participantsSize, participantsOffset int) ([]int64, error)
	GetParticipantIdsBatch(ctx context.Context, chatIds []int64, participantsSize int) ([]*ParticipantIds, error)
	IterateOverChatParticipantIds(ctx context.Context, chatId int64, consumer func(participantIds []int64) error) error
	IterateOverAllParticipantIds(ctx context.Context, consumer func(participantIds []int64) error) error
	IterateOverCoChattedParticipantIds(ctx context.Context, participantId int64, consumer func(participantIds []int64) error) error
	GetParticipantsCount(ctx context.Context, chatId int64) (int, error)
	IsAdmin(ctx context.Context, userId int64, chatId int64) (bool, error)
	IsAdminBatch(ctx context.Context, userId int64, chatIds []int64) (map[int64]bool, error)
	IsAdminBatchByParticipants(ctx context.Context, userIds []int64, chatId int64) ([]UserAdminDbDTO, error)
	IsParticipant(ctx context.Context, userId int64, chatId int64) (bool, error)
	GetChat(ctx context.Context, participantId, chatId int64) (*Chat, error)
	GetChatWithParticipants(ctx context.Context, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error)
	GetChatWithoutParticipants(ctx context.Context, behalfParticipantId, chatId int64) (*ChatWithParticipants, error)
	GetParticipantsCountBatch(ctx context.Context, chatIds []int64) (map[int64]int, error)
	GetMessage(ctx context.Context, chatId int64, userId int64, messageId int64) (*Message, error)
	GetUnreadMessagesCount(ctx context.Context, chatId int64, userId int64) (int64, error)
	SetAdmin(ctx context.Context, userId int64, chatId int64, newAdmin bool) error
	GetChatBasic(ctx context.Context, chatId int64) (*BasicChatDto, error)
	GetChatsBasic(ctx context.Context, chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error)
	GetBlogPostsByLimitOffset(ctx context.Context, reverse bool, limit int, offset int) ([]*Blog, error)
	GetBlogPostsByChatIds(ctx context.Context, ids []int64) ([]*BlogPost, error)
	GetMessageBasic(ctx context.Context, chatId int64, messageId int64) (*MessageBasic, error)
	GetChatsWithParticipants(ctx context.Context, participantId int64, limit int, startingFromItemId *int64, reverse, hasHash bool, searchString string, additionalFoundUserIds []int64, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error)
	CountChatsPerUser(ctx context.Context, userId int64) (int64, error)
	FlipReaction(ctx context.Context, userId int64, chatId int64, messageId int64, reaction string) (bool, error)
	GetChatIds(ctx context.Context, chatsSize, chatsOffset int) ([]int64, error)
	GetPublishedMessagesCount(ctx context.Context, chatId int64) (int64, error)
	GetPinnedMessagesCount(ctx context.Context, chatId int64) (int64, error)
}

func (dbR *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*dbP.Rows, error) {
	return dbR.DB.QueryContext(ctx, query, args...)
}

func (txR *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*dbP.Rows, error) {
	return txR.Tx.QueryContext(ctx, query, args...)
}

func (dbR *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *dbP.Row {
	return dbR.DB.QueryRowContext(ctx, query, args...)
}

func (txR *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *dbP.Row {
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
func Open(conninfo string, maxOpen int, maxIdle int, maxLifetime time.Duration) (*DB, error) {
	if db, err := sql.Open(postgresDriverString, conninfo); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		db.SetConnMaxLifetime(maxLifetime)
		db.SetMaxIdleConns(maxIdle)
		db.SetMaxOpenConns(maxOpen)
		return &DB{db}, nil
	}
}

// Begin starts an returns a new transaction.
func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	if tx, err := db.DB.BeginTx(ctx, nil); err != nil {
		return nil, eris.Wrap(err, "error during interacting with db")
	} else {
		return &Tx{tx}, nil
	}
}

func (tx *Tx) SafeRollback() {
	if err0 := tx.Rollback(); err0 != nil {
		Logger.Errorf("Error during rollback tx %v", err0)
	}
}

//go:embed migrations
var embeddedFiles embed.FS

func migrateInternal(db *sql.DB, path, migrationTable string) {
	staticDir := http.FS(embeddedFiles)
	src, err := httpfs.New(staticDir, "migrations"+path)
	if err != nil {
		Logger.Fatal(err)
	}

	d, err := time.ParseDuration("15m")
	if err != nil {
		Logger.Fatal(err)
	}

	pgInstance, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable:  migrationTable,
		StatementTimeout: d,
	})
	if err != nil {
		Logger.Fatal(err)
	}

	m, err := migrate.NewWithInstance("httpfs", src, "", pgInstance)
	if err != nil {
		Logger.Fatal(err)
	}
	//defer m.Close()
	if err := m.Up(); err != nil && err.Error() != "no change" {
		Logger.Fatal(err)
	}
}

func (db *DB) Migrate(migrationsConfig *MigrationsConfig) {
	Logger.Infof("Starting prod migration")
	migrateInternal(db.DB, "/prod", "go_migrate")
	Logger.Infof("Migration successful prod completed")

	if migrationsConfig.AppendTestData {
		Logger.Infof("Starting test migration")
		migrateInternal(db.DB, "/test", "go_migrate_test")
		Logger.Infof("Migration successful test completed")
	}
}

func ConfigureDb(lc fx.Lifecycle) (*DB, error) {
	dbConnectionString := viper.GetString("postgresql.url")
	maxOpen := viper.GetInt("postgresql.maxOpenConnections")
	maxIdle := viper.GetInt("postgresql.maxIdleConnections")
	maxLifeTime := viper.GetDuration("postgresql.maxLifetime")
	dbInstance, err := Open(dbConnectionString, maxOpen, maxIdle, maxLifeTime)

	if lc != nil {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				Logger.Infof("Stopping db connection")
				return dbInstance.Close()
			},
		})
	}

	return dbInstance, err
}

func (db *DB) RecreateDb() {
	_, err := db.Exec(`
	DROP SCHEMA IF EXISTS public CASCADE;
	CREATE SCHEMA IF NOT EXISTS public;
    GRANT ALL ON SCHEMA public TO chat;
    GRANT ALL ON SCHEMA public TO public;
    COMMENT ON SCHEMA public IS 'standard public schema';
`)
	Logger.Warn("Recreating database")
	if err != nil {
		Logger.Panicf("Error during dropping db: %v", err)
	}
}
