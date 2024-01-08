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
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	TLSEnable  bool     `json:"tls_enable"`
	SASLEnable bool     `json:"sasl_enable"`
	Brokers    []string `json:"brokers"`
}

func (s *KafkaProducerService) Name() ServiceName {
	return s.name
}

func (s *KafkaProducerService) Type() ServiceType {
	return "kafka-producer"
}

func (s *KafkaProducerService) Produce(ctx context.Context, topic string, msg string) error {
	m := &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(msg)}
	_, _, err := s.producer.SendMessage(m)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to send message", zap.Error(err))
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

	if pkg.IsDatadogEnabled() {
		producer = saramatrace.WrapSyncProducer(c, producer)
	}

	kafkaService.producer = producer

	return &kafkaService, nil
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
