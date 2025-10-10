package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/type_registry"
	"time"

	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
)

const EventsFanoutExchange = "async-events-exchange"
const NotificationsFanoutExchange = "notifications-exchange"
const ChatInternalExchange = "chat-internal-exchange"
const AaaEventsExchange = "aaa-profile-events-exchange"
const correlationIdName = "correlationId"

func (rp *RabbitOutputEventsPublisher) Publish(ctx context.Context, correlationId *string, aDto interface{}) error {
	if !rp.enabled {
		return nil
	}

	headers := myRabbitmq.InjectAMQPHeaders(ctx)
	if correlationId != nil {
		headers[correlationIdName] = *correlationId
	}

	aType := rp.typeRegistry.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		rp.lgr.ErrorContext(ctx, "Failed during marshal dto", logger.AttributeError, err)
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
		Headers:      headers,
	}

	publisherName := "output"

	if rp.cfg.RabbitMQ.Dump {
		strData := string(bytea)
		var correlationIdStr string
		if correlationId != nil {
			correlationIdStr = *correlationId
		}
		if rp.cfg.RabbitMQ.PrettyLog && !rp.cfg.Logger.Json {
			fmt.Printf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, correlationId=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, correlationIdStr, strData)
		} else {
			rp.lgr.InfoContext(ctx, fmt.Sprintf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, correlationId=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, correlationIdStr, strData))
		}
	}

	if err := rp.channel.Publish(EventsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.ErrorContext(ctx, "Error during publishing dto", logger.AttributeError, err)
		return err
	} else {
		return nil
	}
}

type RabbitOutputEventsPublisher struct {
	channel      *rabbitmq.Channel
	lgr          *logger.LoggerWrapper
	typeRegistry *type_registry.TypeRegistryInstance
	enabled      bool
	cfg          *config.AppConfig
}

func NewRabbitOutputEventsPublisher(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, typeRegistry *type_registry.TypeRegistryInstance, cfg *config.AppConfig) (*RabbitOutputEventsPublisher, error) {
	cha, err := myRabbitmq.CreateRabbitMqChannel(lgr, connection)
	if err != nil {
		return nil, err
	}
	p := &RabbitOutputEventsPublisher{
		channel:      cha,
		lgr:          lgr,
		typeRegistry: typeRegistry,
		enabled:      false, // events are disabled during fast-forwarding
		cfg:          cfg,
	}

	p.enabled = !cfg.RabbitMQ.SkipPublishOutputEventsOnRewind

	return p, nil
}

func EnableOutputEvents(p *RabbitOutputEventsPublisher) {
	p.enabled = true
}

func (rp *RabbitInternalEventsPublisher) Publish(ctx context.Context, aDto interface{}) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	aType := rp.typeRegistry.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		rp.lgr.ErrorContext(ctx, "Failed during marshal dto", logger.AttributeError, err)
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
		Headers:      headers,
	}

	publisherName := "internal"

	if rp.cfg.RabbitMQ.Dump {
		strData := string(bytea)
		if rp.cfg.RabbitMQ.PrettyLog && !rp.cfg.Logger.Json {
			fmt.Printf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, strData)
		} else {
			rp.lgr.InfoContext(ctx, fmt.Sprintf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, strData))
		}
	}

	if err := rp.channel.Publish(ChatInternalExchange, "", false, false, msg); err != nil {
		rp.lgr.ErrorContext(ctx, "Error during publishing dto", logger.AttributeError, err)
		return err
	} else {
		return nil
	}
}

type RabbitInternalEventsPublisher struct {
	channel      *rabbitmq.Channel
	lgr          *logger.LoggerWrapper
	typeRegistry *type_registry.TypeRegistryInstance
	cfg          *config.AppConfig
}

func NewRabbitInternalEventsPublisher(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, typeRegistry *type_registry.TypeRegistryInstance, cfg *config.AppConfig) (*RabbitInternalEventsPublisher, error) {
	cha, err := myRabbitmq.CreateRabbitMqChannel(lgr, connection)
	if err != nil {
		return nil, err
	}
	return &RabbitInternalEventsPublisher{
		channel:      cha,
		lgr:          lgr,
		typeRegistry: typeRegistry,
		cfg:          cfg,
	}, nil
}

func (rp *RabbitTestInputEventsPublisher) Publish(ctx context.Context, aDto interface{}) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	aType := rp.typeRegistry.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		rp.lgr.ErrorContext(ctx, "Failed during marshal dto", logger.AttributeError, err)
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
		Headers:      headers,
	}

	publisherName := "test"

	if rp.cfg.RabbitMQ.Dump {
		strData := string(bytea)
		if rp.cfg.RabbitMQ.PrettyLog && !rp.cfg.Logger.Json {
			fmt.Printf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, strData)
		} else {
			rp.lgr.InfoContext(ctx, fmt.Sprintf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, strData))
		}
	}

	if err := rp.channel.Publish(AaaEventsExchange, "", false, false, msg); err != nil {
		rp.lgr.ErrorContext(ctx, "Error during publishing dto", logger.AttributeError, err)
		return err
	} else {
		return nil
	}
}

type RabbitTestInputEventsPublisher struct {
	channel      *rabbitmq.Channel
	lgr          *logger.LoggerWrapper
	typeRegistry *type_registry.TypeRegistryInstance
	cfg          *config.AppConfig
}

func NewRabbitTestInputEventsPublisher(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, typeRegistry *type_registry.TypeRegistryInstance, cfg *config.AppConfig) (*RabbitTestInputEventsPublisher, error) {
	cha, err := myRabbitmq.CreateRabbitMqChannel(lgr, connection)
	if err != nil {
		return nil, err
	}
	return &RabbitTestInputEventsPublisher{
		channel:      cha,
		lgr:          lgr,
		typeRegistry: typeRegistry,
		cfg:          cfg,
	}, nil
}

func (rp *RabbitNotificationEventsPublisher) Publish(ctx context.Context, correlationId *string, aDto interface{}) error {
	if !rp.enabled {
		return nil
	}

	headers := myRabbitmq.InjectAMQPHeaders(ctx)
	if correlationId != nil {
		headers[correlationIdName] = *correlationId
	}

	aType := rp.typeRegistry.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		rp.lgr.ErrorContext(ctx, "Failed during marshal dto", logger.AttributeError, err)
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
		Headers:      headers,
	}

	publisherName := "notification"

	if rp.cfg.RabbitMQ.Dump {
		strData := string(bytea)
		var correlationIdStr string
		if correlationId != nil {
			correlationIdStr = *correlationId
		}
		if rp.cfg.RabbitMQ.PrettyLog && !rp.cfg.Logger.Json {
			fmt.Printf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, correlationId=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, correlationIdStr, strData)
		} else {
			rp.lgr.InfoContext(ctx, fmt.Sprintf("[rabbitmq publisher] Sending message: publisher=%s, trace_id=%s, headers=%v, type=%v, correlationId=%v, body: %v\n", publisherName, logger.GetTraceId(ctx), msg.Headers, aType, correlationIdStr, strData))
		}
	}

	if err := rp.channel.Publish(NotificationsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.ErrorContext(ctx, "Error during publishing dto", logger.AttributeError, err)
		return err
	} else {
		return nil
	}
}

type RabbitNotificationEventsPublisher struct {
	channel      *rabbitmq.Channel
	lgr          *logger.LoggerWrapper
	typeRegistry *type_registry.TypeRegistryInstance
	enabled      bool
	cfg          *config.AppConfig
}

func NewRabbitNotificationEventsPublisher(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, typeRegistry *type_registry.TypeRegistryInstance, cfg *config.AppConfig) (*RabbitNotificationEventsPublisher, error) {
	cha, err := myRabbitmq.CreateRabbitMqChannel(lgr, connection)
	if err != nil {
		return nil, err
	}
	p := &RabbitNotificationEventsPublisher{
		channel:      cha,
		lgr:          lgr,
		typeRegistry: typeRegistry,
		cfg:          cfg,
	}

	p.enabled = !cfg.RabbitMQ.SkipPublishNotificationEventsOnRewind

	return p, nil
}

func EnableNotificationEvents(p *RabbitNotificationEventsPublisher) {
	p.enabled = true
}
