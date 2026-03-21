package storage

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "backend-pretest-ai/config"
)

type R2Client struct {
	client     *s3.Client
	bucketName string
	publicURL  string
}

var R2 *R2Client

func InitR2() {
	cfg := appconfig.Cfg.R2

	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: r2Endpoint}, nil
			}),
		),
	)
	if err != nil {
		log.Fatalf("[r2] failed to load config: %v", err)
	}

	R2 = &R2Client{
		client:     s3.NewFromConfig(awsCfg),
		bucketName: cfg.BucketName,
		publicURL:  cfg.PublicURL,
	}

	log.Println("[r2] client initialized successfully")
}

// UploadFile upload file ke R2, return public URL
func (r *R2Client) UploadFile(ctx context.Context, file multipart.File, filename string, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(filename),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return fmt.Sprintf("%s/%s", r.publicURL, filename), nil
}

// DeleteFile hapus file dari R2
func (r *R2Client) DeleteFile(ctx context.Context, filename string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
