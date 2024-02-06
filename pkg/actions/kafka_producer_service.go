package actions

import (
	"context"
	"crypto/tls"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"time"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	saramatrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/IBM/sarama.v1"
	"gopkg.in/DataDog/dd-trace-go.v1/datastreams"
	"gopkg.in/DataDog/dd-trace-go.v1/datastreams/options"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Message struct {
	Value []byte
}

type KafkaProducerService struct {
	name     ServiceName
	config   KafkaProducerServiceConfig
	producer sarama.SyncProducer
}

type KafkaProducerServiceConfig struct {
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	TLSEnable          bool     `json:"tls_enable"`
	SASLEnable         bool     `json:"sasl_enable"`
	Brokers            []string `json:"brokers"`
	TracingServiceName string   `json:"tracing_service_name"`
}

func (s *KafkaProducerService) Name() ServiceName {
	return s.name
}

func (s *KafkaProducerService) Type() ServiceType {
	return "kafka-producer"
}

// NewOTelInterceptor processes span for intercepted messages and add some
// headers with the span data.
//func NewOTelInterceptor(brokers []string) *OTelInterceptor {
//	oi := OTelInterceptor{}
//	oi.tracer = sdktrace.NewTracerProvider().Tracer("chaosmania/kafka-producer")
//
//	// These are based on the spec, which was reachable as of 2020-05-15
//	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/messaging.md
//	oi.fixedAttrs = []attribute.KeyValue{
//		attribute.String(string(semconv.MessagingDestinationNameKey), "topic"),
//		attribute.String("span.otel.kind", "PRODUCER"),
//		attribute.String("messaging.system", "kafka"),
//		attribute.String("net.transport", "IP.TCP"),
//		attribute.String("messaging.url", strings.Join(brokers, ",")),
//	}
//	return &oi
//}

func getProducerMsgSize(msg *sarama.ProducerMessage) (size int64) {
	for _, header := range msg.Headers {
		size += int64(len(header.Key) + len(header.Value))
	}
	if msg.Value != nil {
		size += int64(msg.Value.Length())
	}
	if msg.Key != nil {
		size += int64(msg.Key.Length())
	}
	return size
}

func (s *KafkaProducerService) ddProduce(ctx context.Context, topic string, msg string) error {
	span, _ := tracer.SpanFromContext(ctx)

	m := &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(msg)}

	carrier := saramatrace.NewProducerMessageCarrier(m)
	err := tracer.Inject(span.Context(), carrier)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to inject span", zap.Error(err))
		return err
	}

	setProduceCheckpoint(m)

	partition, offset, err := s.producer.SendMessage(m)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to send message", zap.Error(err))
		//span.Finish(tracer.WithError(err))
		return err
	}

	tracer.TrackKafkaProduceOffset(topic, partition, offset)

	return nil
}

func (s *KafkaProducerService) Produce(ctx context.Context, topic string, msg string) error {
	if pkg.IsDatadogEnabled() {
		return s.ddProduce(ctx, topic, msg)
	}

	prodTracer := otel.GetTracerProvider().Tracer("kafka-producer")

	ctx, span := prodTracer.Start(ctx, "Produce Message", oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
	defer span.End()

	m := &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(msg)}

	span.SetAttributes(attribute.String("span.Kind", "producer"))
	span.SetAttributes(semconv.MessagingKafkaDestinationPartition(int(m.Partition)))
	span.SetAttributes(semconv.MessagingKafkaMessageOffset(int(m.Offset)))
	span.SetAttributes(semconv.MessagingDestinationName(topic))
	span.SetAttributes(semconv.MessagingSystem("kafka"))
	span.SetAttributes(semconv.ServiceName(s.config.TracingServiceName))

	// https://opentelemetry.io/docs/specs/semconv/messaging/kafka/#:~:text=For%20Apache%20Kafka%20producers
	span.SetAttributes(semconv.PeerService(s.config.TracingServiceName))

	setProduceCheckpoint(m)

	_, _, err := s.producer.SendMessage(m)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to send message", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func NewKafkaProducerService(name ServiceName, config map[string]any) (Service, error) {
	cfg, err := pkg.ParseConfig[KafkaProducerServiceConfig](config)
	if err != nil {
		return nil, err
	}

	kafkaService := KafkaProducerService{
		config: *cfg,
		name:   name,
	}
	c := sarama.NewConfig()
	c.Net.DialTimeout = 10 * time.Second

	if cfg.SASLEnable {
		c.Net.SASL.Enable = true
		c.Net.SASL.User = cfg.Username
		c.Net.SASL.Password = cfg.Password
		c.Net.SASL.Mechanism = "PLAIN"
	}

	if cfg.TLSEnable {
		c.Net.TLS.Enable = cfg.TLSEnable
		c.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: true,
			ClientAuth:         0,
		}
	}

	c.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(cfg.Brokers, c)
	if err != nil {
		return nil, err
	}

	//if pkg.IsDatadogEnabled() {
	producer = saramatrace.WrapSyncProducer(c,
		producer,
		saramatrace.WithServiceName(cfg.TracingServiceName))
	//}

	kafkaService.producer = producer

	return &kafkaService, nil
}

func setProduceCheckpoint(msg *sarama.ProducerMessage) {
	edges := []string{"direction:out", "topic:" + msg.Topic, "type:kafka"}
	carrier := saramatrace.NewProducerMessageCarrier(msg)
	ctx, ok := tracer.SetDataStreamsCheckpointWithParams(
		datastreams.ExtractFromBase64Carrier(context.Background(), carrier),
		options.CheckpointParams{PayloadSize: getProducerMsgSize(msg)}, edges...)
	if !ok {
		return
	}

	datastreams.InjectToBase64Carrier(ctx, carrier)
}

func init() {
	SERVICE_TPES["kafka-producer"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewKafkaProducerService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
