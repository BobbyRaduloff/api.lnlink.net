package api_server

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/experiments"
	"api.lnlink.net/src/pkg/models/user"
	"api.lnlink.net/src/pkg/services/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

	return io.ReadAll(result.Body)
}

// Helper function to add a file to a zip archive
func addFileToZip(zipWriter *zip.Writer, filename string, data []byte) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func CreateExperiment(c *gin.Context) {
	userID := GetUserID(c)
	user := user.GetUserByID(userID)

	if user.ModelType != "innocent" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model type"})
		return
	}

	// Parse form data
	name := c.PostForm("name")
	micronsPerPixelStr := c.PostForm("micronsPerPixel")
	micronsPerPixel, err := strconv.ParseFloat(micronsPerPixelStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid micronsPerPixel value"})
		return
	}

	// Get files from multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files provided"})
		return
	}

	// Check total size of all files (1GB max)
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}
	if totalSize > 1<<30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Total size of all files exceeds 1GB limit"})
		return
	}

	// Configure AWS S3 client
	cfg, err := config.LoadDefaultConfig(c.Request.Context(),
		config.WithRegion(global.S3_REGION),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			global.S3_ACCESS_KEY_ID,
			global.S3_SECRET_ACCESS_KEY,
			"",
		)),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure AWS"})
		return
	}

	s3Client := s3.NewFromConfig(cfg)

	// Process each file
	var uploadedFiles []string
	var experimentIDs []string
	for _, file := range files {
		// Check file type
		ext := filepath.Ext(file.Filename)
		if ext == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File %s has no extension", file.Filename)})
			return
		}

		// Open the file
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		defer src.Close()

		// Reset reader for upload
		src.Seek(0, io.SeekStart)

		// Generate unique ID for this experiment
		experimentID := uuid.New().String()
		experimentIDs = append(experimentIDs, experimentID)

		// Generate filename using the experiment ID
		newFilename := fmt.Sprintf("%s%s", experimentID, ext)
		s3Key := fmt.Sprintf("innocent/%s", newFilename)

		// Upload to S3
		_, err = s3Client.PutObject(c.Request.Context(), &s3.PutObjectInput{
			Bucket: aws.String(global.S3_INPUT_BUCKET_NAME),
			Key:    aws.String(s3Key),
			Body:   src,
		})
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to S3"})
			return
		}

		uploadedFiles = append(uploadedFiles, s3Key)
	}

	// Process each uploaded file
	var responses []*models.InnocentResponse
	for i, inputFile := range uploadedFiles {
		// Create experiment with input parameters
		requestBody := models.InnocentInputParams{
			S3InputBucketName:    global.S3_INPUT_BUCKET_NAME,
			S3InputFilePath:      inputFile,
			S3OutputBucketName:   global.S3_OUTPUT_BUCKET_NAME,
			S3OutputMaskFilePath: fmt.Sprintf("innocent/%s.png", experimentIDs[i]),
			S3OutputResultsPath:  fmt.Sprintf("innocent/%s.json", experimentIDs[i]),
			S3OutputTablePath:    fmt.Sprintf("innocent/%s.xlsx", experimentIDs[i]),
			NRays:                32,
			MicronsPerPixel:      micronsPerPixel,
		}

		// Make request to processing service
		response, err := models.InnocentMakeRequest(requestBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process experiment"})
			return
		}

		responses = append(responses, response)
	}

	exps := []experiments.Experiment{}
	for i, response := range responses {
		exps = append(exps, experiments.Experiment{
			FileID:        experimentIDs[i],
			FileExtension: filepath.Ext(files[i].Filename),
			RunpodID:      response.ID,
			Status:        experiments.ExperimentInProgress,
		})
	}
	exp := experiments.MultiExperiment{
		UserID:      userID,
		Experiments: exps,
	}
	err = exp.Create(userID, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create experiment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Experiments created successfully",
		"name":      name,
		"responses": responses,
	})
}

func GetExperiments(c *gin.Context) {
	userID := GetUserID(c)

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	// Get experiments
	experiments, total, err := experiments.GetExperiments(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get experiments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"experiments": experiments,
		"total":       total,
		"page":        page,
		"pageSize":    pageSize,
	})
}

func GetExperimentDownloadLink(c *gin.Context) {
	userID := GetUserID(c)

	// Get experiment ID from URL
	experimentID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid experiment ID"})
		return
	}

	// Verify experiment belongs to user
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(experiments.MultiExperimentCollection)
	var experiment experiments.MultiExperiment
	err = collection.FindOne(context.Background(), bson.M{
		"_id":    experimentID,
		"userId": userID,
	}).Decode(&experiment)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Experiment not found"})
		return
	}

	// Configure AWS S3 client
	cfg, err := config.LoadDefaultConfig(c.Request.Context(),
		config.WithRegion(global.S3_REGION),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			global.S3_ACCESS_KEY_ID,
			global.S3_SECRET_ACCESS_KEY,
			"",
		)),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure AWS"})
		return
	}

	s3Client := s3.NewFromConfig(cfg)

	// Create a zip file in memory
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Create a channel to collect downloaded files
	type fileData struct {
		name string
		data []byte
	}
	fileChan := make(chan fileData, 10)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	// Function to download a file and add it to the channel
	downloadFile := func(key, name string) {
		defer wg.Done()
		data, err := downloadFromS3(s3Client, global.S3_OUTPUT_BUCKET_NAME, key)
		if err != nil {
			// Skip missing files instead of failing
			if strings.Contains(err.Error(), "NoSuchKey") {
				return
			}
			select {
			case errChan <- fmt.Errorf("failed to download %s: %v", key, err):
			default:
			}
			return
		}
		fileChan <- fileData{name: name, data: data}
	}

	// Start downloading files concurrently
	for _, exp := range experiment.Experiments {
		// Download mask files (_0 and _1)
		for i := 0; i <= 1; i++ {
			maskKey := fmt.Sprintf("innocent/%s_%d.png", exp.FileID, i)
			wg.Add(1)
			go downloadFile(maskKey, fmt.Sprintf("%s_mask_%d.png", exp.FileID, i))
		}

		// Download results file
		resultsKey := fmt.Sprintf("innocent/%s.json", exp.FileID)
		wg.Add(1)
		go downloadFile(resultsKey, fmt.Sprintf("%s_results.json", exp.FileID))

		// Download table files (_0 and _1)
		for i := 0; i <= 1; i++ {
			tableKey := fmt.Sprintf("innocent/%s_%d.xlsx", exp.FileID, i)
			wg.Add(1)
			go downloadFile(tableKey, fmt.Sprintf("%s_table_%d.xlsx", exp.FileID, i))
		}
	}

	// Wait for all downloads to complete and close the channel
	go func() {
		wg.Wait()
		close(fileChan)
	}()

	// Add downloaded files to the zip
	for file := range fileChan {
		if err := addFileToZip(zipWriter, file.name, file.data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add %s to zip", file.name)})
			return
		}
	}

	// Check for any errors
	select {
	case err := <-errChan:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	default:
	}

	// Close the zip writer
	if err := zipWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close zip writer"})
		return
	}

	// Upload the zip file to S3
	zipKey := fmt.Sprintf("downloads/%s.zip", experimentID.Hex())
	_, err = s3Client.PutObject(c.Request.Context(), &s3.PutObjectInput{
		Bucket: aws.String(global.S3_OUTPUT_BUCKET_NAME),
		Key:    aws.String(zipKey),
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload zip file"})
		return
	}

	// Generate presigned URL for the zip file
	presignClient := s3.NewPresignClient(s3Client)
	presignResult, err := presignClient.PresignGetObject(c.Request.Context(), &s3.GetObjectInput{
		Bucket: aws.String(global.S3_OUTPUT_BUCKET_NAME),
		Key:    aws.String(zipKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour * 24 // URL valid for 24 hours
	})
	if err != nil {
		log.Printf("Failed to generate presigned URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate presigned URL"})
		return
	}

	log.Printf("Generated presigned URL: %s", presignResult.URL)
	log.Printf("URL expires at: %s", time.Now().Add(24*time.Hour).Format(time.RFC3339))

	c.JSON(http.StatusOK, gin.H{
		"downloadUrl": presignResult.URL,
	})
}

func RegisterExperimentRoutes(router *gin.Engine) {
	router.POST("/api/experiments", AuthMiddleware(), CreateExperiment)
	router.GET("/api/experiments", AuthMiddleware(), GetExperiments)
	router.GET("/api/experiments/:id/download", AuthMiddleware(), GetExperimentDownloadLink)
}
