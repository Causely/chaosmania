package actions

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"
	"time"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type KafkaConsumerService struct {
	name   BackgroundServiceName
	config *KafkaConsumerServiceConfig
	cfg    *sarama.Config
}

type KafkaConsumerServiceConfig struct {
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	TLSEnable  bool     `json:"tls_enable"`
	SASLEnable bool     `json:"sasl_enable"`
	Brokers    []string `json:"brokers"`
	Topic      string   `json:"topic"`
	Script     string   `json:"script"`
}

func (s *KafkaConsumerService) Name() BackgroundServiceName {
	return s.name
}

func (s *KafkaConsumerService) Type() BackgroundServiceType {
	return "kafka-consumer"
}

func (s *KafkaConsumerService) Run(ctx context.Context) error {
	client, err := sarama.NewConsumerGroup(s.config.Brokers, "group1", s.cfg)
	if err != nil {
		return err
	}

	defer client.Close()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, []string{s.config.Topic}, s); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}

				logger.FromContext(ctx).Warn("failed to consume message", zap.Error(err))
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
		}
	}()

	wg.Wait()

	return nil
}

func NewKafkaConsumerService(name BackgroundServiceName, config map[string]any) (BackgroundService, error) {
	cfg, err := pkg.ParseConfig[KafkaConsumerServiceConfig](config)
	if err != nil {
		return nil, err
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

	kafkaService := KafkaConsumerService{
		name:   name,
		config: cfg,
		cfg:    c,
	}

	return &kafkaService, nil
}

func init() {
	BACKGROUND_SERVICE_TPES["kafka-consumer"] = func(name BackgroundServiceName, m map[string]any) BackgroundService {
		s, err := NewKafkaConsumerService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}

func (consumer *KafkaConsumerService) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *KafkaConsumerService) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (consumer *KafkaConsumerService) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/IBM/sarama/blob/main/consumer_group.go#L27-L29
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			consumer.handleMessage(session.Context(), message)
			session.MarkMessage(message, "")
		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}

func (consumer *KafkaConsumerService) handleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	cfg := ScriptConfig{
		Script:  consumer.config.Script,
		Message: string(message.Value),
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["Script"].Execute(ctx, c)
}
