package actions

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type RabbitMQConsumerService struct {
	name   BackgroundServiceName
	config *RabbitMQConsumerServiceConfig
	conn   *amqp.Connection
}

type RabbitMQConsumerServiceConfig struct {
	URL           string `json:"url"`
	Queue         string `json:"queue"`
	Script        string `json:"script"`
	PeerService   string `json:"peer_service"`
	PeerNamespace string `json:"peer_namespace"`
}

func (consumer *RabbitMQConsumerService) Name() BackgroundServiceName {
	return consumer.name
}

func (consumer *RabbitMQConsumerService) Type() BackgroundServiceType {
	return "rabbitmq-consumer"
}

func (consumer *RabbitMQConsumerService) getChannel() (*amqp.Channel, error) {
	if consumer.conn != nil && consumer.conn.IsClosed() {
		consumer.conn = nil
	}

	if consumer.conn == nil {
		conn, err := amqp.Dial(consumer.config.URL)
		if err != nil {
			return nil, err
		}

		consumer.conn = conn
	}

	return consumer.conn.Channel()
}

func (consumer *RabbitMQConsumerService) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			err := consumer.run(ctx)
			if err != nil {
				logger.FromContext(ctx).Warn("failed to consume message", zap.Error(err))
			}
		}
	}
}

func (consumer *RabbitMQConsumerService) run(ctx context.Context) error {
	ch, err := consumer.getChannel()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to get channel", zap.Error(err))
		return err
	}

	// Declare a queue for receiving
	q, err := ch.QueueDeclare(
		consumer.config.Queue, // name
		false,                 // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // arguments
	)

	if err != nil {
		logger.FromContext(ctx).Warn("failed to declare queue", zap.Error(err))
		return err
	}

	deliveries, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	if err != nil {
		logger.FromContext(ctx).Warn("failed to consume message", zap.Error(err))
		return err
	}

	for msg := range deliveries {
		consumer.handleMessage(ctx, msg)
	}

	return nil
}

func (consumer *RabbitMQConsumerService) ddHandleMessage(ctx context.Context, msg amqp.Delivery) error {
	span := tracer.StartSpan("handle_message",
		tracer.ResourceName("handle_message"))
	defer span.Finish()

	// This span is just to show the relationship between the kafka consumer and the topic.
	child := tracer.StartSpan("rabbitmq.consume",
		tracer.ResourceName("Consume message"),
		tracer.Tag("queue", consumer.config.Queue),      // Required tag
		tracer.Tag("span.kind", "consumer"),             // Required tag
		tracer.ServiceName(consumer.config.PeerService), // Required tag
		tracer.ChildOf(span.Context()),
	)
	child.Finish()

	err := tracer.Inject(span.Context(), ctx)
	if err != nil {
		msg.Ack(false)
		return err
	}

	cfg := ScriptConfig{
		Script:  consumer.config.Script,
		Message: string(msg.Body),
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		msg.Ack(false)
		child.Finish(tracer.WithError(err))
		return err
	}

	err = ACTIONS["Script"].Execute(ctx, c)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to execute script", zap.Error(err))
	}

	return nil
}

func (consumer *RabbitMQConsumerService) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	if pkg.IsDatadogEnabled() {
		return consumer.ddHandleMessage(ctx, msg)
	}
	consumeTracer := otel.GetTracerProvider().Tracer("rabbitmq-consumer")
	ctx, span := consumeTracer.Start(ctx, "Consume Queue "+consumer.config.Queue, oteltrace.WithSpanKind(oteltrace.SpanKindConsumer))
	defer span.End()
	span.SetAttributes(semconv.MessagingDestinationName(consumer.config.Queue))
	span.SetAttributes(semconv.MessagingSystemRabbitmq)

	// https://opentelemetry.io/docs/specs/semconv/messaging/kafka/#:~:text=For%20Apache%20Kafka%20producers
	span.SetAttributes(semconv.PeerService(consumer.config.PeerService))
	span.SetAttributes(attribute.String("peer.namespace", consumer.config.PeerNamespace))

	cfg := ScriptConfig{
		Script:  consumer.config.Script,
		Message: string(msg.Body),
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		msg.Ack(false)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else {
		span.SetStatus(codes.Ok, "message received")
	}

	err = ACTIONS["Script"].Execute(ctx, c)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to execute script", zap.Error(err))
	}

	return nil
}

func NewRabbitMQConsumerService(name BackgroundServiceName, config map[string]any) (BackgroundService, error) {
	cfg, err := pkg.ParseConfig[RabbitMQConsumerServiceConfig](config)
	if err != nil {
		return nil, err
	}

	rabbitService := RabbitMQConsumerService{
		name:   name,
		config: cfg,
	}

	return &rabbitService, nil
}

func init() {
	BACKGROUND_SERVICE_TYPES["rabbitmq-consumer"] = func(name BackgroundServiceName, m map[string]any) BackgroundService {
		s, err := NewRabbitMQConsumerService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
