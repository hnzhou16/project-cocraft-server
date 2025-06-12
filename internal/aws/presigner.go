package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type Presigner struct {
	PresignClient *s3.PresignClient
	S3Client      *s3.Client
	Bucket        string
}

func NewPresigner(accessKey, secretKey, region, bucket string) (*Presigner, error) {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(creds)),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(awsCfg)
	presigner := s3.NewPresignClient(s3Client)
	return &Presigner{
		PresignClient: presigner,
		S3Client:      s3Client,
		Bucket:        bucket,
	}, nil
}

func (p *Presigner) EnsureUserFolderExists(ctx context.Context, bucketName, userID string) error {
	folderKey := fmt.Sprintf("%s/", userID)

	// Check if the folder already exists by listing objects
	listResp, err := p.S3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(folderKey),
		MaxKeys: aws.Int32(1), // Only need to check if something exists
	})
	if err != nil {
		return fmt.Errorf("error checking folder existence for %s: %v", userID, err)
	}

	// If folder does not exist, create a placeholder object
	if len(listResp.Contents) == 0 {
		_, err := p.S3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(folderKey + "placeholder.txt"),
			Body:   nil, // Empty body just to create the folder
		})
		if err != nil {
			return fmt.Errorf("error creating placeholder for folder %s: %v", folderKey, err)
		}
	}

	return nil
}

// GetImageURL generates a presigned URL for retrieving an object.
func (p *Presigner) GetImageURL(ctx context.Context, objectKey string, exp time.Duration) (string, error) {
	req, err := p.PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = exp
	})
	if err != nil {
		return "", fmt.Errorf("Couldn't get a presigned request to get %v:%v. Error: %v\n", p.Bucket, objectKey, err)
	}
	return req.URL, err
}

// GenerateUploadURL generates a presigned URL for uploading an object.
func (p *Presigner) GenerateUploadURL(ctx context.Context, userID, ext string, exp time.Duration) (*v4.PresignedHTTPRequest, string, error) {
	objectKey := fmt.Sprintf("user_uploads/%s/%s.%s", userID, uuid.New().String(), ext)

	// Generate presigned PUT URL
	req, err := p.PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.Bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(ext),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = exp
	})

	if err != nil {
		return nil, "", err
	}

	return req, objectKey, nil
}

func (p *Presigner) DeleteImage(ctx context.Context, objectKey string) error {
	_, err := p.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return err
	}

	return nil
}
