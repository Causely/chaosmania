package actions

import (
	"context"
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
	Endpoint           string        `json:"endpoint"`
	AccessKeyID        string        `json:"accesskeyid"`
	SecretAccessKey    string        `json:"secretaccesskey"`
	UseSSL             bool          `json:"usessl"`
	Buckets            []MinioBucket `json:"buckets"`
	TracingServiceName string        `json:"tracing_service_name"`
}

func (s *MinioService) Name() ServiceName {
	return s.name
}

func (s *MinioService) Type() ServiceType {
	return "minio"
}

func (s *MinioService) startSpan(ctx context.Context, resource string) tracer.Span {
	span, _ := tracer.SpanFromContext(ctx)

	return tracer.StartSpan("minio.command",
		tracer.ResourceName(resource),
		tracer.ServiceName(s.config.TracingServiceName),
		tracer.ChildOf(span.Context()),
		tracer.Tag(ext.SpanKind, ext.SpanKindClient),
		tracer.Tag("out.host", s.config.Endpoint),
	)
}

func (s *MinioService) Get_object(ctx context.Context, bucket string, object string) (string, error) {
	child := s.startSpan(ctx, "GetObject")
	defer child.Finish()

	obj, err := s.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		child.Finish(tracer.WithError(err))
		return "", err
	}

	data, err := io.ReadAll(obj)
	if err != nil {
		child.Finish(tracer.WithError(err))
		return "", err
	}

	return string(data), nil
}

func (s *MinioService) Put_object(ctx context.Context, bucket string, object string, data string) error {
	child := s.startSpan(ctx, "PutObject")
	defer child.Finish()

	length := len(data)
	reader := strings.NewReader(data)
	_, err := s.client.PutObject(ctx, bucket, object, reader, int64(length), minio.PutObjectOptions{ContentType: "text/plain"})
	if err != nil {
		child.Finish(tracer.WithError(err))
		return err
	}

	return nil
}

func (s *MinioService) Remove_object(ctx context.Context, bucket string, object string) error {
	child := s.startSpan(ctx, "RemoveObject")
	defer child.Finish()

	err := s.client.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
	if err != nil {
		child.Finish(tracer.WithError(err))
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
	SERVICE_TPES["minio"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewMinioService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
