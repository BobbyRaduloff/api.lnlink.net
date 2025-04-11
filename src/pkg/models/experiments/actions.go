package experiments

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"api.lnlink.net/src/pkg/global"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (exp *MultiExperiment) Create(userID primitive.ObjectID, name string) error {
	exp.ID = primitive.NewObjectID()
	exp.UserID = userID
	exp.Name = name

	exp.CreatedAt = time.Now()

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(MultiExperimentCollection)
	_, err := collection.InsertOne(context.Background(), exp)
	if err != nil {
		return err
	}

	return nil
}

func (exp *MultiExperiment) AddExperiment(experiment Experiment) error {
	exp.Experiments = append(exp.Experiments, experiment)
	return nil
}

// GetExperiments retrieves paginated experiments for a user
func GetExperiments(userID primitive.ObjectID, page, pageSize int) ([]MultiExperiment, int64, error) {
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(MultiExperimentCollection)

	// Get total count
	total, err := collection.CountDocuments(context.Background(), bson.M{"userId": userID})
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	skip := int64((page - 1) * pageSize)
	cursor, err := collection.Find(context.Background(),
		bson.M{"userId": userID},
		options.Find().
			SetSkip(skip).
			SetLimit(int64(pageSize)).
			SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.Background())

	var experiments []MultiExperiment
	if err = cursor.All(context.Background(), &experiments); err != nil {
		return nil, 0, err
	}

	return experiments, total, nil
}

// GenerateDownloadLink creates a presigned URL for downloading experiment results
func GenerateDownloadLink(experimentID primitive.ObjectID) (string, error) {
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(MultiExperimentCollection)

	var experiment MultiExperiment
	err := collection.FindOne(context.Background(), bson.M{"_id": experimentID}).Decode(&experiment)
	if err != nil {
		return "", err
	}

	// Configure AWS S3 client
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(global.S3_REGION),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			global.S3_ACCESS_KEY_ID,
			global.S3_SECRET_ACCESS_KEY,
			"",
		)),
	)
	if err != nil {
		return "", err
	}

	s3Client := s3.NewFromConfig(cfg)

	// Create a zip file containing all experiment results
	zipKey := fmt.Sprintf("downloads/%s.zip", experimentID.Hex())

	// Create a new zip file in memory
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add each experiment's output files to the zip
	for _, exp := range experiment.Experiments {
		// Add mask files (_0 and _1)
		for i := 0; i <= 1; i++ {
			maskKey := fmt.Sprintf("innocent/%s_%d.png", exp.FileID, i)
			maskData, err := downloadFromS3(s3Client, global.S3_OUTPUT_BUCKET_NAME, maskKey)
			if err == nil {
				if err := addFileToZip(zipWriter, fmt.Sprintf("%s_mask_%d.png", exp.FileID, i), maskData); err != nil {
					return "", fmt.Errorf("failed to add mask file to zip: %v", err)
				}
			}
		}

		// Add results file
		resultsKey := fmt.Sprintf("innocent/%s.json", exp.FileID)
		resultsData, err := downloadFromS3(s3Client, global.S3_OUTPUT_BUCKET_NAME, resultsKey)
		if err == nil {
			if err := addFileToZip(zipWriter, fmt.Sprintf("%s_results.json", exp.FileID), resultsData); err != nil {
				return "", fmt.Errorf("failed to add results file to zip: %v", err)
			}
		}

		// Add table files (_0 and _1)
		for i := 0; i <= 1; i++ {
			tableKey := fmt.Sprintf("innocent/%s_%d.xlsx", exp.FileID, i)
			tableData, err := downloadFromS3(s3Client, global.S3_OUTPUT_BUCKET_NAME, tableKey)
			if err == nil {
				if err := addFileToZip(zipWriter, fmt.Sprintf("%s_table_%d.xlsx", exp.FileID, i), tableData); err != nil {
					return "", fmt.Errorf("failed to add table file to zip: %v", err)
				}
			}
		}
	}

	// Close the zip writer before uploading
	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close zip writer: %v", err)
	}

	// Upload the zip file to S3
	_, err = s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(global.S3_OUTPUT_BUCKET_NAME),
		Key:         aws.String(zipKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/zip"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload zip file: %v", err)
	}

	// Generate presigned URL for the zip file
	presignClient := s3.NewPresignClient(s3Client)
	presignResult, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(global.S3_OUTPUT_BUCKET_NAME),
		Key:    aws.String(zipKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour * 24 // URL valid for 24 hours
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	return presignResult.URL, nil
}

// Helper function to download a file from S3
func downloadFromS3(s3Client *s3.Client, bucket, key string) ([]byte, error) {
	result, err := s3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %v", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object: %v", err)
	}
	return data, nil
}

// Helper function to add a file to a zip archive
func addFileToZip(zipWriter *zip.Writer, filename string, data []byte) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %v", err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write to zip: %v", err)
	}
	return nil
}
