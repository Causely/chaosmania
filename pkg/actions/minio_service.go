package actions

import (
	"context"
	"io"
	"strings"

	"github.com/Causely/chaosmania/pkg"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
}

func (s *MinioService) Name() ServiceName {
	return s.name
}

func (s *MinioService) Type() ServiceType {
	return "minio"
}

func (s *MinioService) Get_object(ctx context.Context, bucket string, object string) (string, error) {
	obj, err := s.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(obj)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *MinioService) Put_object(ctx context.Context, bucket string, object string, data string) error {
	length := len(data)
	reader := strings.NewReader(data)
	_, err := s.client.PutObject(ctx, bucket, object, reader, int64(length), minio.PutObjectOptions{ContentType: "text/plain"})
	return err
}

func (s *MinioService) Remove_object(ctx context.Context, bucket string, object string) error {
	return s.client.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
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
