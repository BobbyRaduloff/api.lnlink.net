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
	"sync"

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

// Helper function to create a zip file containing experiment results
func createExperimentZip(s3Client *s3.Client, experiment experiments.MultiExperiment) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	var missingFiles []string
	var includedFiles []string

	// Use a mutex to protect concurrent writes to the zip writer and file lists
	var mu sync.Mutex

	// Create a buffered channel to limit concurrent downloads
	sem := make(chan struct{}, 10) // Allow up to 10 concurrent downloads

	// Create a wait group to wait for all downloads to complete
	var wg sync.WaitGroup

	// Create error channel to collect errors
	errChan := make(chan error, 1)
	done := make(chan bool)

	for _, exp := range experiment.Experiments {
		log.Printf("Processing experiment with FileID: %s", exp.FileID)

		// Download files concurrently
		fileTypes := []struct {
			key      string
			filename string
			isArray  bool
		}{
			{fmt.Sprintf("innocent/%s.json", exp.FileID), fmt.Sprintf("%s_results.json", exp.FileID), false},
		}

		// Add mask and table files (both _0 and _1)
		for i := 0; i <= 1; i++ {
			maskKey := fmt.Sprintf("innocent/%s_%d.png", exp.FileID, i)
			tableKey := fmt.Sprintf("innocent/%s_%d.xlsx", exp.FileID, i)
			fileTypes = append(fileTypes,
				struct {
					key, filename string
					isArray       bool
				}{
					maskKey, fmt.Sprintf("%s_mask_%d.png", exp.FileID, i), true,
				},
				struct {
					key, filename string
					isArray       bool
				}{
					tableKey, fmt.Sprintf("%s_table_%d.xlsx", exp.FileID, i), true,
				},
			)
		}

		for _, ft := range fileTypes {
			wg.Add(1)
			go func(key, filename string, isArray bool) {
				defer wg.Done()

				// Acquire semaphore
				sem <- struct{}{}
				defer func() { <-sem }()

				if data, err := downloadFromS3(s3Client, global.S3_OUTPUT_BUCKET_NAME, key); err == nil {
					mu.Lock()
					if err := addFileToZip(zipWriter, filename, data); err != nil {
						select {
						case errChan <- fmt.Errorf("failed to add file %s: %v", filename, err):
						default:
						}
					} else {
						desc := "results file"
						if isArray {
							desc = filepath.Ext(filename)[1:] + " file"
						}
						includedFiles = append(includedFiles, fmt.Sprintf("%s for %s", desc, exp.FileID))
						log.Printf("Added %s to zip", filename)
					}
					mu.Unlock()
				} else {
					mu.Lock()
					missingFiles = append(missingFiles, fmt.Sprintf("%s for %s", filepath.Base(key), exp.FileID))
					log.Printf("Failed to download file %s: %v", key, err)
					mu.Unlock()
				}
			}(ft.key, ft.filename, ft.isArray)
		}
	}

	// Wait for downloads in a goroutine
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for either completion or error
	select {
	case err := <-errChan:
		return nil, err
	case <-done:
		// All downloads completed successfully
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %v", err)
	}

	// Only return an error if no files were included at all
	if len(includedFiles) == 0 {
		return nil, fmt.Errorf("no files were found in S3")
	}

	log.Printf("Successfully created zip file with %d files. Size: %d bytes", len(includedFiles), buf.Len())
	if len(missingFiles) > 0 {
		log.Printf("Warning: Some files were missing from S3: %v", missingFiles)
	}

	return buf, nil
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
	for i, file := range files {
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

		// Generate unique filename while preserving extension
		newFilename := fmt.Sprintf("%d%s", i+1, ext)
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
	var experimentIDs []string
	for _, inputFile := range uploadedFiles {
		// Generate unique ID for this experiment
		experimentID := uuid.New().String()

		// Create experiment with input parameters
		requestBody := models.InnocentInputParams{
			S3Region:             global.S3_REGION,
			S3AccessKeyID:        global.S3_ACCESS_KEY_ID,
			S3AccessKeySecret:    global.S3_SECRET_ACCESS_KEY,
			S3InputBucketName:    global.S3_INPUT_BUCKET_NAME,
			S3InputFilePath:      inputFile,
			S3OutputBucketName:   global.S3_OUTPUT_BUCKET_NAME,
			S3OutputMaskFilePath: fmt.Sprintf("innocent/%s.png", experimentID),
			S3OutputResultsPath:  fmt.Sprintf("innocent/%s.json", experimentID),
			S3OutputTablePath:    fmt.Sprintf("innocent/%s.xlsx", experimentID),
			S3ModelBucketName:    global.S3_MODEL_BUCKET_NAME,
			ModelNames:           []string{"innocent-cells", "innocent-lipids"},
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
		experimentIDs = append(experimentIDs, experimentID)
	}

	exps := []experiments.Experiment{}
	for i, response := range responses {
		exps = append(exps, experiments.Experiment{
			FileID:          experimentIDs[i],
			RunpodID:        response.ID,
			Status:          experiments.ExperimentInProgress,
			RetryCount:      0,
			MicronsPerPixel: micronsPerPixel,
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

	// Create the zip file in memory
	zipBuf, err := createExperimentZip(s3Client, experiment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create zip file: %v", err)})
		return
	}

	// Get the complete zip data
	zipData := zipBuf.Bytes()
	contentLength := len(zipData)

	// Set response headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", experimentID.Hex()))
	c.Header("Content-Length", fmt.Sprintf("%d", contentLength))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Connection", "Keep-Alive")
	c.Header("Cache-Control", "no-cache")

	// Write the complete zip data
	c.Data(http.StatusOK, "application/zip", zipData)
}

func RegisterExperimentRoutes(router *gin.Engine) {
	router.POST("/api/experiments", AuthMiddleware(), CreateExperiment)
	router.GET("/api/experiments", AuthMiddleware(), GetExperiments)
	router.GET("/api/experiments/:id/download", AuthMiddleware(), GetExperimentDownloadLink)
}
