package actions

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type RabbitMQProducerService struct {
	name    ServiceName
	config  RabbitMQProducerServiceConfig
	conn    *amqp.Connection
	channel *amqp.Channel
}

type RabbitMQProducerServiceConfig struct {
	URL           string `json:"url"`
	PeerService   string `json:"peer_service"`
	PeerNamespace string `json:"peer_namespace"`
}

func (producer *RabbitMQProducerService) Name() ServiceName {
	return producer.name
}

func (producer *RabbitMQProducerService) Type() ServiceType {
	return "rabbitmq-producer"
}

func (producer *RabbitMQProducerService) getChannel() (*amqp.Channel, error) {
	if producer.conn != nil && producer.conn.IsClosed() {
		producer.conn = nil
		producer.channel = nil
	}

	if producer.conn == nil {
		conn, err := amqp.Dial(producer.config.URL)
		if err != nil {
			return nil, err
		}

		producer.conn = conn
		channel, err := conn.Channel()
		if err != nil {
			return nil, err
		}

		producer.channel = channel
	}

	return producer.channel, nil
}

func (producer *RabbitMQProducerService) Close() error {
	if producer.conn != nil {
		return producer.conn.Close()
	}

	producer.conn = nil
	producer.channel = nil
	return nil
}

func (producer *RabbitMQProducerService) ddProduce(ctx context.Context, queue string, msg string) error {
	ch, err := producer.getChannel()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to get channel", zap.Error(err))
		return err
	}

	// Declare a queue for sending
	q, err := ch.QueueDeclare(
		queue, // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		logger.FromContext(ctx).Warn("failed to declare queue", zap.Error(err))

		if err := producer.Close(); err != nil {
			logger.FromContext(ctx).Warn("failed to close connection", zap.Error(err))
		}

		return err
	}

	span := tracer.StartSpan("rabbitmq.produce",
		tracer.ResourceName("Produce message"),
		tracer.Tag("queue", queue))
	defer span.Finish()

	child := tracer.StartSpan("rabbitmq.produce",
		tracer.ResourceName("Produce message"),
		tracer.Tag("queue", queue),
		tracer.Tag("span.kind", "producer"),
		tracer.ServiceName(producer.config.PeerService),
		tracer.ChildOf(span.Context()),
	)
	defer child.Finish()

	// Send a message
	return ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		},
	)
}

func (producer *RabbitMQProducerService) Produce(ctx context.Context, queue string, msg string) error {
	if pkg.IsDatadogEnabled() {
		return producer.ddProduce(ctx, queue, msg)
	}
	ch, err := producer.getChannel()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to get channel", zap.Error(err))
		return err
	}

	// Declare a queue for sending
	q, err := ch.QueueDeclare(
		queue, // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		logger.FromContext(ctx).Warn("failed to declare queue", zap.Error(err))

		if err := producer.Close(); err != nil {
			logger.FromContext(ctx).Warn("failed to close connection", zap.Error(err))
		}

		return err
	}

	prodTracer := otel.GetTracerProvider().Tracer("kafka-producer")

	_, span := prodTracer.Start(ctx, "Produce Message", oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
	defer span.End()

	span.SetAttributes(semconv.MessagingDestinationName(queue))
	span.SetAttributes(semconv.MessagingSystemRabbitmq)

	// https://opentelemetry.io/docs/specs/semconv/messaging/kafka/#:~:text=For%20Apache%20Kafka%20producers
	span.SetAttributes(semconv.PeerService(producer.config.PeerService))
	span.SetAttributes(attribute.String("peer.namespace", producer.config.PeerNamespace))

	// Send a message
	return ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		},
	)
}

func NewRabbitMQProducerService(name ServiceName, config map[string]any) (Service, error) {
	cfg, err := pkg.ParseConfig[RabbitMQProducerServiceConfig](config)
	if err != nil {
		return nil, err
	}

	rabbitService := RabbitMQProducerService{
		config: *cfg,
		name:   name,
	}

	return &rabbitService, nil
}

func init() {
	SERVICE_TYPES["rabbitmq-producer"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewRabbitMQProducerService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
