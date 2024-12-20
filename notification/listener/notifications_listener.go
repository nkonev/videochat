package listener

import (
	"context"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/notification/dto"
	"nkonev.name/notification/logger"
	"nkonev.name/notification/rabbitmq"
	"nkonev.name/notification/services"
)

type NotificationsListener func(*amqp.Delivery) error

func CreateNotificationsListener(service *services.NotificationService, lgr *log.Logger) NotificationsListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "notification.listener")
		defer span.End()

		bytesData := msg.Body
		strData := string(bytesData)
		logger.GetLogEntry(ctx, lgr).Debugf("Received %v", strData)

		var bindTo = new(dto.NotificationEvent)
		err := json.Unmarshal(msg.Body, bindTo)
		if err != nil {
			logger.GetLogEntry(ctx, lgr).Errorf("Unable to unmarshall notification %v", err)
			return err
		}

		service.HandleChatNotification(ctx, bindTo)

		return nil
	}
}
