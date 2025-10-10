package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	"go.uber.org/fx"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
)

func RunScheduler(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	scheduler *dcron.Cron,
	ct *CleanAbandonedChatsTask,
	cd *CleanDeletedUserDataTask,
	lc fx.Lifecycle,
) error {
	scheduler.Start()
	lgr.Info("Scheduler started")

	if cfg.Schedulers.CleanAbandonedChatsTask.Enabled {
		lgr.Info("Adding task " + ct.Key() + " to scheduler")
		err := scheduler.AddJobs(ct)
		if err != nil {
			return err
		}
	} else {
		lgr.Info("Task " + ct.Key() + " is disabled")
	}

	if cfg.Schedulers.CleanDeletedUsersDataTask.Enabled {
		lgr.Info("Adding task " + cd.Key() + " to scheduler")
		err := scheduler.AddJobs(cd)
		if err != nil {
			return err
		}
	} else {
		lgr.Info("Task " + cd.Key() + " is disabled")
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Stopping scheduler")
			<-scheduler.Stop().Done()
			return nil
		},
	})
	return nil
}
