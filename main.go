package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func main() {
	bucketName := flag.String("bucket", "", "S3 bucket name")
	dateStr := flag.String("date", "", "Delete files older than this date (RFC 3339 format or other common formats)")
	workers := flag.Int("workers", 10, "Number of concurrent workers")
	endpoint := flag.String("endpoint", "", "S3 compatible endpoint (e.g., http://localhost:9000 for Minio)")
	region := flag.String("region", "us-east-1", "AWS region or custom region for S3 compatible storage")
	accessKey := flag.String("access-key", "", "Access key for S3 or S3-compatible service")
	secretKey := flag.String("secret-key", "", "Secret key for S3 or S3-compatible service")
	maxKeysPerDelete := flag.Int("max-keys", 1000, "Maximum number of keys to delete in a single DeleteObjects call")
	flag.Parse()

	if *bucketName == "" || *dateStr == "" {
		log.Fatal("Bucket name and date are required")
	}

	targetDate, err := parseDate(*dateStr)
	if err != nil {
		log.Fatalf("Invalid date format: %v", err)
	}

	var client *s3.Client
	if *endpoint != "" {
		// S3-compatible service
		if *accessKey == "" || *secretKey == "" {
			log.Fatal("Access key and secret key are required for S3-compatible services")
		}
		client, err = createS3CompatibleClient(*endpoint, *region, *accessKey, *secretKey)
	} else {
		// AWS S3
		client, err = createAWSS3Client(*region)
	}
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	objectsToDelete := make(chan []types.ObjectIdentifier, *workers)
	var wg sync.WaitGroup
	var totalDeleted int64

	// Start worker goroutines
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(client, *bucketName, objectsToDelete, &wg, &totalDeleted)
	}

	startTime := time.Now()
	totalObjects, err := listAndDeleteObjects(client, *bucketName, targetDate, objectsToDelete, *maxKeysPerDelete)
	if err != nil {
		log.Fatalf("Failed to list and delete objects: %v", err)
	}

	close(objectsToDelete)
	wg.Wait()

	duration := time.Since(startTime)
	log.Printf("Operation complete. Deleted %d/%d objects in %v", totalDeleted, totalObjects, duration)
}

func createS3CompatibleClient(endpoint, region, accessKey, secretKey string) (*s3.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg, err := awsconfig.LoadDefaultConfig(
		context.TODO(),
		awsconfig.WithCredentialsProvider(creds),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	}), nil
}

func createAWSS3Client(region string) (*s3.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return s3.NewFromConfig(cfg), nil
}

func worker(client *s3.Client, bucket string, objectsToDelete <-chan []types.ObjectIdentifier, wg *sync.WaitGroup, totalDeleted *int64) {
	defer wg.Done()

	for objects := range objectsToDelete {
		_, err := client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		})

		if err != nil {
			log.Printf("Failed to delete objects: %v", err)
		} else {
			atomic.AddInt64(totalDeleted, int64(len(objects)))
			fmt.Printf("Deleted %d objects\n", len(objects))
		}
	}
}

func listAndDeleteObjects(client *s3.Client, bucket string, targetDate time.Time, objectsToDelete chan<- []types.ObjectIdentifier, maxKeysPerDelete int) (int, error) {
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})

	var objectsBuffer []types.ObjectIdentifier
	totalObjects := 0

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return totalObjects, err
		}

		for _, obj := range page.Contents {
			if obj.LastModified.Before(targetDate) {
				objectsBuffer = append(objectsBuffer, types.ObjectIdentifier{Key: obj.Key})
				totalObjects++

				if len(objectsBuffer) >= maxKeysPerDelete {
					objectsToDelete <- objectsBuffer
					objectsBuffer = []types.ObjectIdentifier{}
				}
			}
		}
	}

	if len(objectsBuffer) > 0 {
		objectsToDelete <- objectsBuffer
	}

	return totalObjects, nil
}

func parseDate(dateStr string) (time.Time, error) {
	// Try parsing with RFC3339 format first
	t, err := time.Parse(time.RFC3339, dateStr)
	if err == nil {
		return t, nil
	}

	// Try parsing with other common formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		time.RFC822,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
