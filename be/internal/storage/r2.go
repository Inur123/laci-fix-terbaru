package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Object struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
}

type R2 struct {
	client *s3.Client
	bucket string
	ready  bool
}

func NewR2(ctx context.Context, accountID, accessKey, secretKey, bucket string) (*R2, error) {
	if accountID == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return &R2{}, nil
	}
	endpoint := "https://" + accountID + ".r2.cloudflarestorage.com"
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return &R2{client: client, bucket: bucket, ready: true}, nil
}

func (r *R2) Put(ctx context.Context, key, contentType string, value []byte) error {
	return r.PutReader(ctx, key, contentType, bytes.NewReader(value), int64(len(value)))
}

func (r *R2) PutReader(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	if !r.ready {
		return errors.New("R2 configuration missing")
	}
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.bucket), Key: aws.String(key), Body: body, ContentLength: aws.Int64(size), ContentType: aws.String(contentType),
	})
	return err
}

func (r *R2) Get(ctx context.Context, key string) ([]byte, error) {
	body, err := r.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	const maxEncryptedObjectSize = 65 << 20
	value, err := io.ReadAll(io.LimitReader(body, maxEncryptedObjectSize+1))
	if err != nil {
		return nil, err
	}
	if len(value) > maxEncryptedObjectSize {
		return nil, errors.New("encrypted object too large")
	}
	return value, nil
}

func (r *R2) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !r.ready {
		return nil, errors.New("R2 configuration missing")
	}
	result, err := r.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (r *R2) Delete(ctx context.Context, key string) error {
	if !r.ready || key == "" {
		return nil
	}
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key)})
	return err
}

func (r *R2) List(ctx context.Context, prefix string) ([]Object, error) {
	if !r.ready {
		return nil, errors.New("R2 configuration missing")
	}
	objects := make([]Object, 0)
	paginator := s3.NewListObjectsV2Paginator(r.client, &s3.ListObjectsV2Input{Bucket: aws.String(r.bucket), Prefix: aws.String(prefix)})
	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, object := range result.Contents {
			objects = append(objects, Object{Key: aws.ToString(object.Key), Size: aws.ToInt64(object.Size), LastModified: aws.ToTime(object.LastModified)})
		}
	}
	return objects, nil
}

func (r *R2) SignedURL(ctx context.Context, key string, validFor time.Duration) (string, error) {
	if !r.ready {
		return "", errors.New("R2 configuration missing")
	}
	presigner := s3.NewPresignClient(r.client)
	result, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key)}, s3.WithPresignExpires(validFor))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
