package tasks

import (
	postgresLock "github.com/nkonev/dcron/plugin/lock/postgres_pgx5"
	"nkonev.name/chat/db"

	"github.com/nkonev/dcron"
	"nkonev.name/chat/logger"
)

func Scheduler(dba *db.DB, lgr *logger.LoggerWrapper) (*dcron.Cron, error) {
	return dcron.NewCron(
		postgresLock.WithPool(dba.Pool),
		dcron.WithSLog(lgr),
	), nil
}
