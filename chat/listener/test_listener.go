package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sanity-io/litter"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/chat/config"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/type_registry"
	"slices"
	"time"
)

type TestEventAccumulator struct {
	cfg          *config.AppConfig
	lgr          *logger.LoggerWrapper
	eventsBuffer []any
}

type TestOutputEventAccumulator struct {
	TestEventAccumulator
}

func (p *TestEventAccumulator) OnEvent(ctx context.Context, e any) {
	p.eventsBuffer = append(p.eventsBuffer, e)
}

func NewRabbitTestOutputEventAccumulator(cfg *config.AppConfig, lgr *logger.LoggerWrapper) *TestOutputEventAccumulator {
	return &TestOutputEventAccumulator{
		TestEventAccumulator{
			cfg:          cfg,
			lgr:          lgr,
			eventsBuffer: make([]any, 0),
		},
	}
}

type TestNotificationEventAccumulator struct {
	TestEventAccumulator
}

func NewRabbitTestNotificationEventAccumulator(cfg *config.AppConfig, lgr *logger.LoggerWrapper) *TestNotificationEventAccumulator {
	return &TestNotificationEventAccumulator{
		TestEventAccumulator{
			cfg:          cfg,
			lgr:          lgr,
			eventsBuffer: make([]any, 0),
		},
	}
}

func (p *TestEventAccumulator) Clean() {
	p.eventsBuffer = []any{}
}

// there can be more events than asserters

// AssertHasEventsOrdered returns true if all the asserters are matched events in order of asserters
func (p *TestEventAccumulator) AssertHasEventsOrdered(asserters []func(e any) bool) bool {
	j := 0 // both second pointer and num of success comparisons

	for _, e := range p.eventsBuffer {
		if j >= len(asserters) { // bound check
			break
		}

		if asserters[j](e) {
			p.lgr.Info("Ordered - satisfying asserter with index", "index", j)
			j++
		}
	}

	return j == len(asserters)
}

// AssertHasEventsUnordered returns true if all the asserters are matched events in any order
func (p *TestEventAccumulator) AssertHasEventsUnordered(asserters []func(e any) bool) bool {
	assertersCopy := make([]func(e any) bool, len(asserters))
	copy(assertersCopy, asserters)

	for eventIdx, e := range p.eventsBuffer {
		for j, c := range assertersCopy {
			if c(e) {
				p.lgr.Info("Unordered - satisfying asserter for event with index", "index", eventIdx)
				assertersCopy = slices.Delete(assertersCopy, j, j+1)
				break // inner loop
			}
		}
	}

	return len(assertersCopy) == 0
}

func (p *TestEventAccumulator) AwaitForBufferContainsSpecifiedEvents(duration time.Duration, ordered bool, comparators []func(e any) bool) error {
	du := p.cfg.RabbitMQ.CheckAreEventsProcessedInterval

	startTime := time.Now()

	for {
		currTime := time.Now()
		if startTime.Add(duration).Before(currTime) {
			return fmt.Errorf("timeout error, there no specified events in %v", duration)
		}

		p.lgr.Info("Checking condition, the buffer is")
		if p.cfg.RabbitMQ.DumpTestAccumulator {
			litter.Dump(p.eventsBuffer)
		}

		fv := func() bool {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("panic occured: ", r)
				}
			}()
			if ordered {
				if p.AssertHasEventsOrdered(comparators) {
					p.lgr.Info("Buffer contains the specified events, exiting successfully")
					return true
				}
			} else {
				if p.AssertHasEventsUnordered(comparators) {
					p.lgr.Info("Buffer contains the specified events, exiting successfully")
					return true
				}
			}

			return false
		}()
		if fv {
			return nil // success exit
		}

		time.Sleep(du)
	}

}

type TestOutputEventListener func(*amqp.Delivery) error

func CreateRabbitTestOutputEventListener(service *TestOutputEventAccumulator, lgr *logger.LoggerWrapper, typeRegistry *type_registry.TypeRegistryInstance) TestOutputEventListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "test.output.event.listener")
		defer span.End()

		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type

		lgr.DebugContext(ctx, "Received", "data", strData, "type", aType)

		if !typeRegistry.HasType(aType) {
			lgr.ErrorContext(ctx, "Unexpected type in rabbit test_output_listener", "type", aType)
			return nil
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.GlobalUserEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				lgr.ErrorContext(ctx, "Error during deserialize notification", logger.AttributeError, err)
				return err
			}
			service.OnEvent(ctx, &bindTo)
		case dto.ChatEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				lgr.ErrorContext(ctx, "Error during deserialize notification", logger.AttributeError, err)
				return err
			}
			service.OnEvent(ctx, &bindTo)
		default:
			lgr.ErrorContext(ctx, "Unexpected type:", "instance", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))
		}

		return nil
	}
}

type TestNotificationEventListener func(*amqp.Delivery) error

func CreateRabbitTestNotificationEventListener(service *TestNotificationEventAccumulator, lgr *logger.LoggerWrapper, typeRegistry *type_registry.TypeRegistryInstance) TestNotificationEventListener {
	tr := otel.Tracer("amqp/listener")

	return func(msg *amqp.Delivery) error {
		ctx := rabbitmq.ExtractAMQPHeaders(context.Background(), msg.Headers)
		ctx, span := tr.Start(ctx, "test.notification.event.listener")
		defer span.End()

		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type

		lgr.DebugContext(ctx, "Received", "data", strData, "type", aType)

		if !typeRegistry.HasType(aType) {
			lgr.ErrorContext(ctx, "Unexpected type in rabbit test_notification_listener", "type", aType)
			return nil
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.NotificationEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				lgr.ErrorContext(ctx, "Error during deserialize notification", logger.AttributeError, err)
				return err
			}
			service.OnEvent(ctx, &bindTo)
		default:
			lgr.ErrorContext(ctx, "Unexpected type:", "instance", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))
		}

		return nil
	}
}
