package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/chat/config"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/services"
	"nkonev.name/chat/type_registry"
)

type AaaUserProfileUpdateListener func(*amqp.Delivery) error

func CreateRabbitAaaUserProfileUpdateListener(lgr *logger.LoggerWrapper, cfg *config.AppConfig, not *services.InputEventHandler, typeRegistry *type_registry.TypeRegistryInstance) AaaUserProfileUpdateListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "aaa.listener")
		defer span.End()

		bytesData := msg.Body
		aType := msg.Type

		listenerName := "aaa"

		if cfg.RabbitMQ.Dump {
			strData := string(bytesData)

			if cfg.RabbitMQ.PrettyLog && !cfg.Logger.Json {
				fmt.Printf("[rabbitmq listener] Received message: listener=%s, trace_id=%s, headers=%v, type=%v, body: %v\n", listenerName, logger.GetTraceId(ctx), msg.Headers, aType, strData)
			} else {
				lgr.InfoContext(ctx, fmt.Sprintf("[rabbitmq listener] Received message: listener=%s, trace_id=%s, headers=%v, type=%v, body: %v\n", listenerName, logger.GetTraceId(ctx), msg.Headers, aType, strData))
			}
		}

		if !typeRegistry.HasType(aType) {
			lgr.InfoContext(ctx, "Unexpected type in rabbit aaa_listener", "type", aType)
			return nil
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.UserAccountEventChanged:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				lgr.ErrorContext(ctx, "Error during deserialize notification", logger.AttributeError, err)
				return err
			}
			if bindTo.EventType == dto.EventTypeUserAccountChanged {
				not.NotifyAboutProfileChanged(ctx, bindTo.User)
			}

		default:
			lgr.ErrorContext(ctx, "Unexpected type:", "instance", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))
		}

		return nil
	}
}
