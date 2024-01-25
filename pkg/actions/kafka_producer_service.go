package actions

import (
	"context"
	"crypto/tls"
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

func (s *KafkaProducerService) Produce(ctx context.Context, topic string, msg string) error {
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
		return err
	}

	if pkg.IsDatadogEnabled() {
		tracer.TrackKafkaProduceOffset(topic, partition, offset)
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

	if pkg.IsDatadogEnabled() {
		producer = saramatrace.WrapSyncProducer(c,
			producer,
			saramatrace.WithServiceName(cfg.TracingServiceName))
	}

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
