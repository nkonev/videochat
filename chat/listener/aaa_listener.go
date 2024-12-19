package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/services"
	"nkonev.name/chat/type_registry"
)

type AaaUserProfileUpdateListener func(*amqp.Delivery) error

func CreateAaaUserProfileUpdateListener(lgr *log.Logger, not *services.Events, typeRegistry *type_registry.TypeRegistryInstance, db *db.DB) AaaUserProfileUpdateListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "aaa.listener")
		defer span.End()

		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type
		lgr.Debugf("Received %v with type %v", strData, aType)

		if !typeRegistry.HasType(aType) {
			errStr := fmt.Sprintf("Unexpected type in rabbit fanout notifications: %v", aType)
			lgr.Debugf(errStr)
			return nil
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.UserAccountEventChanged:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				lgr.Errorf("Error during deserialize notification %v", err)
				return err
			}
			if bindTo.EventType == "user_account_changed" {
				not.NotifyAboutProfileChanged(ctx, bindTo.User, db)
			}

		default:
			lgr.Errorf("Unexpected type : %v", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))
		}

		return nil
	}
}
