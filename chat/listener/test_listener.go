package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/sanity-io/litter"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"nkonev.name/chat/config"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/type_registry"
)

type TestEventAccumulator struct {
	cfg          *config.AppConfig
	lgr          *logger.LoggerWrapper
	eventsBuffer chan any
}

type TestOutputEventAccumulator struct {
	TestEventAccumulator
}

func (p *TestEventAccumulator) OnEvent(ctx context.Context, e any) {
	p.eventsBuffer <- e
}

func newBufferedChan() chan any {
	return make(chan any, 4096)
}

func NewRabbitTestOutputEventAccumulator(cfg *config.AppConfig, lgr *logger.LoggerWrapper) *TestOutputEventAccumulator {
	return &TestOutputEventAccumulator{
		TestEventAccumulator{
			cfg:          cfg,
			lgr:          lgr,
			eventsBuffer: newBufferedChan(),
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
			eventsBuffer: newBufferedChan(),
		},
	}
}

func (p *TestEventAccumulator) Clean() {
	p.reset()
}

func (p *TestEventAccumulator) reset() {
	for {
		select {
		case <-p.eventsBuffer:
			// just drain
		default:
			return
		}
	}
}

// there can be more events than asserters

// assertHasEventsOrdered returns true if all the asserters are matched events in order of asserters
func (p *TestEventAccumulator) assertHasEventsOrdered(ctx context.Context, asserters []func(e any) bool) error {
	matched := 0

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out ordered: %w", ctx.Err())
		case e, ok := <-p.eventsBuffer:
			if !ok {
				return errors.New("buffer is closed")
			}

			if p.cfg.RabbitMQ.DumpTestAccumulator {
				litter.Dump(e)
			}

			if matched > len(asserters) { // bound check
				return errors.New("matched is greater than len(asserters), it should never happen")
			}

			if asserters[matched](e) {
				p.lgr.Info("Ordered - satisfying asserter with index", "index", matched)
				matched++

				if matched == len(asserters) {
					return nil // ok
				}
			}
		}
	}
}

// assertHasEventsUnordered returns true if all the asserters are matched events in any order
func (p *TestEventAccumulator) assertHasEventsUnordered(ctx context.Context, asserters []func(e any) bool) error {
	remainingAsserters := make([]func(e any) bool, len(asserters))
	copy(remainingAsserters, asserters)

	eventIdx := 0

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out unordered: %w", ctx.Err())
		case e, ok := <-p.eventsBuffer:
			if !ok {
				return errors.New("buffer is closed")
			}

			if p.cfg.RabbitMQ.DumpTestAccumulator {
				litter.Dump(e)
			}

			for j, asserter := range remainingAsserters {
				if asserter(e) {
					p.lgr.Info("Unordered - satisfying asserter for event with index", "index", eventIdx)
					remainingAsserters = slices.Delete(remainingAsserters, j, j+1)

					if len(remainingAsserters) == 0 {
						return nil // ok
					}

					break // inner loop
				}
			}

			eventIdx++
		}
	}
}

func (p *TestEventAccumulator) AwaitForBufferContainsSpecifiedEvents(duration time.Duration, ordered bool, comparators []func(e any) bool) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	fve := func() error {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("panic occured: ", r)
			}
		}()
		if ordered {
			return p.assertHasEventsOrdered(ctx, comparators)
		} else {
			return p.assertHasEventsUnordered(ctx, comparators)
		}
	}()
	return fve
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
