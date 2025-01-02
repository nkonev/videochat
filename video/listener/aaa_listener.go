package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/video/dto"
	"nkonev.name/video/logger"
	"nkonev.name/video/rabbitmq"
	"nkonev.name/video/services"
	"nkonev.name/video/type_registry"
)

type AaaUserProfileUpdateListener func(*amqp.Delivery) error

func CreateAaaUserSessionsKilledListener(lgr *logger.Logger, userService *services.UserService, typeRegistry *type_registry.TypeRegistryInstance) AaaUserProfileUpdateListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "aaa.listener")
		defer span.End()

		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type
		lgr.WithTracing(ctx).Debugf("Received %v with type %v", strData, aType)

		if !typeRegistry.HasType(aType) {
			errStr := fmt.Sprintf("Unexpected type in rabbit fanout notifications: %v", aType)
			lgr.WithTracing(ctx).Debugf(errStr)
			return nil
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.UserSessionsKilledEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				lgr.WithTracing(ctx).Errorf("Error during deserialize notification %v", err)
				return err
			}
			if bindTo.EventType == "user_sessions_killed" {
				userService.KickUser(ctx, bindTo.UserId)
				userService.ProcessCallOnDisabling(ctx, bindTo.UserId)
			}

		default:
			lgr.WithTracing(ctx).Errorf("Unexpected type : %v", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))

		}

		return nil
	}
}
