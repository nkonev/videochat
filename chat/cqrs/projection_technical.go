package cqrs

import (
	"context"
	"fmt"
	"nkonev.name/chat/db"
	"nkonev.name/chat/logger"
)

func (m *CommonProjection) OnTechnicalProjectionsTruncated(ctx context.Context, event *ProjectionsTruncated) error {
	err := db.RunResetDatabaseSoft(m.db, m.cfg)
	if err != nil {
		return fmt.Errorf("Error during resetting: %w", err)
	}

	err = db.RunMigrations(m.db, m.cfg)
	if err != nil {
		return fmt.Errorf("Error during migrating: %w", err)
	}

	err = m.SetIsTruncatingCompleted(ctx)
	if err != nil {
		return fmt.Errorf("Error during set IsTruncatingCompleted: %w", err)
	}

	return nil
}

func (m *CommonProjection) OnTechnicalAbandonedChatRemoved(ctx context.Context, event *TechnicalAbandonedChatRemoved) error {
	has, err := m.HasParticipants(ctx, m.db, []int64{event.ChatId})
	if err != nil {
		return err
	}

	if has[event.ChatId] {
		m.lgr.InfoContext(ctx, "Actually this chat has participants, skipping", logger.AttributeChatId, event.ChatId) // to prevent race condition when sheduler found an empty chat because the ParticipantAdd event still wasn't processed
		return nil
	}

	_, err = m.db.ExecContext(ctx, "delete from chat_common where id = $1", event.ChatId)
	if err != nil {
		return err
	}

	return nil
}
