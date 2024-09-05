package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/montag451/go-eventbus"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/event/dto"
	. "nkonev.name/event/logger"
	"nkonev.name/event/rabbitmq"
	"nkonev.name/event/type_registry"
)

type EventsListener func(*amqp.Delivery) error

func CreateEventsListener(bus *eventbus.Bus, typeRegistry *type_registry.TypeRegistryInstance) EventsListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "event.listener")
		defer span.End()

		traceString := rabbitmq.SerializeValues(ctx)

		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type
		Logger.Debugf("Received %v with type %v", strData, aType)

		if !typeRegistry.HasType(aType) {
			errStr := fmt.Sprintf("Unexpected type in rabbit fanout notifications: %v", aType)
			Logger.Errorf(errStr)
			return errors.New(errStr)
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.ChatEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			bindTo.TraceString = traceString

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}

		case dto.GlobalUserEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			bindTo.TraceString = traceString

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}

		case []dto.UserOnline:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}

			var converted = dto.ArrayUserOnline{
				UserOnlines: bindTo,
				TraceString: traceString,
			}

			err = bus.PublishAsync(converted)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}

		case dto.GeneralEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			bindTo.TraceString = traceString

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}
		case dto.UserAccountEventChanged:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			bindTo.TraceString = traceString

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}
		case dto.UserAccountEventCreated:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			bindTo.TraceString = traceString

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}
		case dto.UserAccountEventDeleted:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			bindTo.TraceString = traceString

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}
		case dto.UserSessionsKilledEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			bindTo.TraceString = traceString

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}

		default:
			Logger.Errorf("Unexpected type : %v", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))
		}

		return nil
	}
}
