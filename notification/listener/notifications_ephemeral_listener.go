package listener

import (
	"context"
	"encoding/json"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/notification/dto"
	"nkonev.name/notification/logger"
	"nkonev.name/notification/rabbitmq"
	"nkonev.name/notification/services"
)

type NotificationsEphemeralListener func(*amqp.Delivery) error

func CreateNotificationsEphemeralListener(service *services.NotificationEphemeralService, lgr *logger.Logger) NotificationsEphemeralListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "notification.ephemeral.listener")
		defer span.End()

		bytesData := msg.Body
		strData := string(bytesData)
		lgr.WithTracing(ctx).Debugf("Received %v", strData)

		var bindTo = new(dto.NotificationEphemeralEvent)
		err := json.Unmarshal(msg.Body, bindTo)
		if err != nil {
			lgr.WithTracing(ctx).Errorf("Unable to unmarshall notification %v", err)
			return err
		}

		service.HandleChatNotification(ctx, bindTo)

		return nil
	}
}
