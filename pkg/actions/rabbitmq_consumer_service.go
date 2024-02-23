package actions

import (
	"context"

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
	URL                string `json:"url"`
	Queue              string `json:"queue"`
	Script             string `json:"script"`
	TracingServiceName string `json:"tracing_service_name"`
}

func (s *RabbitMQConsumerService) Name() BackgroundServiceName {
	return s.name
}

func (s *RabbitMQConsumerService) Type() BackgroundServiceType {
	return "rabbitmq-consumer"
}

func (s *RabbitMQConsumerService) getChannel() (*amqp.Channel, error) {
	if s.conn != nil && s.conn.IsClosed() {
		s.conn = nil
	}

	if s.conn == nil {
		conn, err := amqp.Dial(s.config.URL)
		if err != nil {
			return nil, err
		}

		s.conn = conn
	}

	return s.conn.Channel()
}

func (s *RabbitMQConsumerService) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			err := s.run(ctx)
			if err != nil {
				logger.FromContext(ctx).Warn("failed to consume message", zap.Error(err))
			}
		}
	}
}

func (s *RabbitMQConsumerService) run(ctx context.Context) error {
	ch, err := s.getChannel()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to get channel", zap.Error(err))
		return err
	}

	// Declare a queue for receiving
	q, err := ch.QueueDeclare(
		s.config.Queue, // name
		false,          // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
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
		s.handleMessage(ctx, msg)
	}

	return nil
}

func (s *RabbitMQConsumerService) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	span := tracer.StartSpan("handle_message",
		tracer.ResourceName("handle_message"))
	defer span.Finish()

	// This span is just to show the relationship between the kafka consumer and the topic.
	child := tracer.StartSpan("rabbitmq.consume",
		tracer.ResourceName("Consume message"),
		tracer.Tag("queue", s.config.Queue),             // Required tag
		tracer.Tag("span.kind", "consumer"),             // Required tag
		tracer.ServiceName(s.config.TracingServiceName), // Required tag
		tracer.ChildOf(span.Context()),
	)
	child.Finish()

	err := tracer.Inject(span.Context(), ctx)
	if err != nil {
		msg.Ack(false)
		return err
	}

	cfg := ScriptConfig{
		Script:  s.config.Script,
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
	BACKGROUND_SERVICE_TPES["rabbitmq-consumer"] = func(name BackgroundServiceName, m map[string]any) BackgroundService {
		s, err := NewRabbitMQConsumerService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
