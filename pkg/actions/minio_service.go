package actions

import (
	"context"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"io"
	"strings"

	"github.com/Causely/chaosmania/pkg"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type MinioBucket struct {
	Name string `json:"name"`
}

type MinioService struct {
	name   ServiceName
	config MinioServiceConfig
	client *minio.Client
}

type MinioServiceConfig struct {
	Endpoint        string        `json:"endpoint"`
	AccessKeyID     string        `json:"accesskeyid"`
	SecretAccessKey string        `json:"secretaccesskey"`
	UseSSL          bool          `json:"usessl"`
	Buckets         []MinioBucket `json:"buckets"`
	PeerService     string        `json:"peer_service"`
	PeerNamespace   string        `json:"peer_namespace"`
}

func (ms *MinioService) Name() ServiceName {
	return ms.name
}

func (ms *MinioService) Type() ServiceType {
	return "minio"
}

func (ms *MinioService) ddStartSpan(ctx context.Context, resource string) tracer.Span {
	span, _ := tracer.SpanFromContext(ctx)

	return tracer.StartSpan("minio.command",
		tracer.ResourceName(resource),
		tracer.ServiceName(ms.config.PeerService),
		tracer.ChildOf(span.Context()),
		tracer.Tag(ext.SpanKind, ext.SpanKindClient),
		tracer.Tag("out.host", ms.config.Endpoint),
	)
}

func (ms *MinioService) Get_object(ctx context.Context, bucket string, object string) (string, error) {
	var child tracer.Span
	var span oteltrace.Span
	if pkg.IsDatadogEnabled() {
		child = ms.ddStartSpan(ctx, "GetObject")
		defer child.Finish()
	} else {
		prodTracer := otel.GetTracerProvider().Tracer("minio-storage")

		_, span = prodTracer.Start(ctx, "GetObject", oteltrace.WithSpanKind(oteltrace.SpanKindClient))
		defer span.End()

		span.SetAttributes(semconv.AWSS3Bucket(bucket))
		span.SetAttributes(semconv.PeerService(ms.config.PeerService))
		span.SetAttributes(attribute.String("peer.namespace", ms.config.PeerNamespace))
	}

	obj, err := ms.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		child.Finish(tracer.WithError(err))
		return "", err
	}

	data, err := io.ReadAll(obj)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to send message", zap.Error(err))
		if pkg.IsDatadogEnabled() {
			child.Finish(tracer.WithError(err))
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return "", err
	}

	return string(data), nil
}

func (ms *MinioService) Put_object(ctx context.Context, bucket string, object string, data string) error {
	var child tracer.Span
	var span oteltrace.Span
	if pkg.IsDatadogEnabled() {
		child := ms.ddStartSpan(ctx, "PutObject")
		defer child.Finish()
	} else {
		prodTracer := otel.GetTracerProvider().Tracer("minio-storage")

		_, span = prodTracer.Start(ctx, "PutObject", oteltrace.WithSpanKind(oteltrace.SpanKindClient))
		defer span.End()

		span.SetAttributes(semconv.AWSS3Bucket(bucket))
		span.SetAttributes(semconv.PeerService(ms.config.PeerService))
		span.SetAttributes(attribute.String("peer.namespace", ms.config.PeerNamespace))
	}

	length := len(data)
	reader := strings.NewReader(data)
	_, err := ms.client.PutObject(ctx, bucket, object, reader, int64(length), minio.PutObjectOptions{ContentType: "text/plain"})
	if err != nil {
		logger.FromContext(ctx).Warn("failed to send message", zap.Error(err))
		if pkg.IsDatadogEnabled() {
			child.Finish(tracer.WithError(err))
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	return nil
}

func (ms *MinioService) Remove_object(ctx context.Context, bucket string, object string) error {
	var child tracer.Span
	var span oteltrace.Span
	if pkg.IsDatadogEnabled() {
		child := ms.ddStartSpan(ctx, "RemoveObject")
		defer child.Finish()
	} else {
		prodTracer := otel.GetTracerProvider().Tracer("minio-storage")

		_, span = prodTracer.Start(ctx, "RemoveObject", oteltrace.WithSpanKind(oteltrace.SpanKindClient))
		defer span.End()

		span.SetAttributes(semconv.AWSS3Bucket(bucket))
		span.SetAttributes(semconv.PeerService(ms.config.PeerService))
		span.SetAttributes(attribute.String("peer.namespace", ms.config.PeerNamespace))
	}

	err := ms.client.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
	if err != nil {
		logger.FromContext(ctx).Warn("failed to send message", zap.Error(err))
		if pkg.IsDatadogEnabled() {
			child.Finish(tracer.WithError(err))
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	return nil
}

func NewMinioService(name ServiceName, config map[string]any) (Service, error) {
	cfg, err := pkg.ParseConfig[MinioServiceConfig](config)
	if err != nil {
		return nil, err
	}

	minioService := MinioService{
		config: *cfg,
		name:   name,
	}

	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})

	if err != nil {
		return nil, err
	}

	minioService.client = minioClient

	for _, bucket := range cfg.Buckets {
		exists, err := minioClient.BucketExists(context.Background(), bucket.Name)
		if err != nil {
			return nil, err
		}

		if exists {
			continue
		}

		err = minioClient.MakeBucket(context.Background(), bucket.Name, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}

	return &minioService, nil
}

func init() {
	SERVICE_TYPES["minio"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewMinioService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
