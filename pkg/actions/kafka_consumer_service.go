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
	saramatrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/IBM/sarama.v1"
	"gopkg.in/DataDog/dd-trace-go.v1/datastreams"
	"gopkg.in/DataDog/dd-trace-go.v1/datastreams/options"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type KafkaConsumerService struct {
	name   BackgroundServiceName
	config *KafkaConsumerServiceConfig
	cfg    *sarama.Config
}

type KafkaConsumerServiceConfig struct {
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	TLSEnable          bool     `json:"tls_enable"`
	SASLEnable         bool     `json:"sasl_enable"`
	Brokers            []string `json:"brokers"`
	Topic              string   `json:"topic"`
	Script             string   `json:"script"`
	TracingServiceName string   `json:"tracing_service_name"`
	Group              string   `json:"group"`
}

func (s *KafkaConsumerService) Name() BackgroundServiceName {
	return s.name
}

func (s *KafkaConsumerService) Type() BackgroundServiceType {
	return "kafka-consumer"
}

func (s *KafkaConsumerService) Run(ctx context.Context) error {
	client, err := sarama.NewConsumerGroup(s.config.Brokers, s.config.Group, s.cfg)
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
			err := consumer.handleMessage(session.Context(), message)
			if err != nil {
				logger.FromContext(session.Context()).Warn("failed to handle message", zap.Error(err))
			}

			session.MarkMessage(message, "")
		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}

func getConsumerMsgSize(msg *sarama.ConsumerMessage) (size int64) {
	for _, header := range msg.Headers {
		size += int64(len(header.Key) + len(header.Value))
	}
	return size + int64(len(msg.Value)+len(msg.Key))
}

func (consumer *KafkaConsumerService) handleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "process_message",
		tracer.ResourceName("process_message"),
	)
	defer span.Finish()

	// This span is just to show the relationship between the kafka consumer and the topic.
	child := tracer.StartSpan("kafka.consume",
		tracer.ResourceName("Consume Topic "+consumer.config.Topic),
		tracer.SpanType(ext.SpanTypeMessageConsumer),

		tracer.Tag(ext.MessagingKafkaPartition, message.Partition),
		tracer.Tag("offset", message.Offset),
		tracer.Tag(ext.Component, "IBM/sarama"),
		tracer.Tag(ext.SpanKind, ext.SpanKindConsumer), // Required tag
		tracer.Tag(ext.MessagingSystem, ext.MessagingSystemKafka),
		tracer.Measured(),

		tracer.Tag("topic", consumer.config.Topic),             // Required tag
		tracer.Tag("group", consumer.config.Group),             // Required tag
		tracer.ServiceName(consumer.config.TracingServiceName), // Required tag
		tracer.ChildOf(span.Context()),
	)
	child.Finish()

	setConsumeCheckpoint(true, consumer.config.Group, message)

	cfg := ScriptConfig{
		Script:  consumer.config.Script,
		Message: string(message.Value),
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	err = ACTIONS["Script"].Execute(ctx, c)
	span.Finish(tracer.WithError(err))
	return err
}

func setConsumeCheckpoint(enabled bool, groupID string, msg *sarama.ConsumerMessage) {
	if !enabled || msg == nil {
		return
	}
	edges := []string{"direction:in", "topic:" + msg.Topic, "type:kafka"}
	if groupID != "" {
		edges = append(edges, "group:"+groupID)
	}
	carrier := saramatrace.NewConsumerMessageCarrier(msg)
	ctx, ok := tracer.SetDataStreamsCheckpointWithParams(datastreams.ExtractFromBase64Carrier(context.Background(), carrier), options.CheckpointParams{PayloadSize: getConsumerMsgSize(msg)}, edges...)
	if !ok {
		return
	}
	datastreams.InjectToBase64Carrier(ctx, carrier)
	if groupID != "" {
		// only track Kafka lag if a consumer group is set.
		// since there is no ack mechanism, we consider that messages read are committed right away.
		tracer.TrackKafkaCommitOffset(groupID, msg.Topic, msg.Partition, msg.Offset)
	}
}
