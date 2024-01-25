package actions

import (
	"context"

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
	URL                string `json:"url"`
	TracingServiceName string `json:"tracing_service_name"`
}

func (s *RabbitMQProducerService) Name() ServiceName {
	return s.name
}

func (s *RabbitMQProducerService) Type() ServiceType {
	return "rabbitmq-producer"
}

func (s *RabbitMQProducerService) getChannel() (*amqp.Channel, error) {
	if s.conn != nil && s.conn.IsClosed() {
		s.conn = nil
		s.channel = nil
	}

	if s.conn == nil {
		conn, err := amqp.Dial(s.config.URL)
		if err != nil {
			return nil, err
		}

		s.conn = conn
		channel, err := conn.Channel()
		if err != nil {
			return nil, err
		}

		s.channel = channel
	}

	return s.channel, nil
}

func (s *RabbitMQProducerService) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}

	s.conn = nil
	s.channel = nil
	return nil
}

func (s *RabbitMQProducerService) Produce(ctx context.Context, queue string, msg string) error {
	ch, err := s.getChannel()
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

		if err := s.Close(); err != nil {
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
		tracer.ServiceName(s.config.TracingServiceName),
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
	SERVICE_TPES["rabbitmq-producer"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewRabbitMQProducerService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
