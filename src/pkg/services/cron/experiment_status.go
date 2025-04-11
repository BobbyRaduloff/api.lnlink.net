package cron

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/experiments"
	"api.lnlink.net/src/pkg/models/user"
	"api.lnlink.net/src/pkg/services/models"
	"go.mongodb.org/mongo-driver/bson"
)

const MaxRetries = 3

// UpdateExperimentStatuses checks all in-progress experiments and updates their status
func UpdateExperimentStatuses() error {
	log.Println("[ExperimentStatusCron] Starting experiment status update cycle")
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(experiments.MultiExperimentCollection)

	// Find all experiments that are in progress
	cursor, err := collection.Find(context.Background(), bson.M{
		"experiments": bson.M{
			"$elemMatch": bson.M{
				"status": experiments.ExperimentInProgress,
			},
		},
	})
	if err != nil {
		log.Printf("[ExperimentStatusCron] Error finding in-progress experiments: %v", err)
		return err
	}
	defer cursor.Close(context.Background())

	var multiExperiments []experiments.MultiExperiment
	if err := cursor.All(context.Background(), &multiExperiments); err != nil {
		log.Printf("[ExperimentStatusCron] Error reading experiments from cursor: %v", err)
		return err
	}

	log.Printf("[ExperimentStatusCron] Found %d experiments in progress", len(multiExperiments))

	for _, multiExp := range multiExperiments {
		log.Printf("[ExperimentStatusCron] Processing experiment group ID: %s", multiExp.ID)
		for i, exp := range multiExp.Experiments {
			if exp.Status != experiments.ExperimentInProgress {
				log.Printf("[ExperimentStatusCron] Skipping experiment %d in group %s - not in progress", i, multiExp.ID)
				continue
			}

			log.Printf("[ExperimentStatusCron] Checking status for experiment %d (RunPod ID: %s) in group %s", i, exp.RunpodID, multiExp.ID)

			// Get status from RunPod
			status, err := models.InnocentGetStatus(exp.RunpodID)
			if err != nil {
				log.Printf("[ExperimentStatusCron] Error getting RunPod status for experiment %d (RunPod ID: %s): %v", i, exp.RunpodID, err)
				continue
			}

			log.Printf("[ExperimentStatusCron] RunPod status for experiment %d (RunPod ID: %s): %s", i, exp.RunpodID, status.Status)

			// Update experiment status based on RunPod response
			switch status.Status {
			case "COMPLETED":
				if status.InnocentCompletedResponse != nil {
					log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) completed successfully", i, exp.RunpodID)
					multiExp.Experiments[i].Status = experiments.ExperimentCompleted
					multiExp.Experiments[i].ExecutionTimeMillis = int(math.Ceil(float64(status.InnocentCompletedResponse.ExecutionTime) / 1000))

					// Deduct tokens based on model type
					user := user.GetUserByID(multiExp.UserID)
					if user != nil {
						tokensToDeduct := 8 // Default 8 tokens per image
						if user.ModelType == "innocent" {
							tokensToDeduct = 16 // 16 tokens per image for innocent model
						}
						user.AddTokens(-tokensToDeduct)
						log.Printf("[ExperimentStatusCron] Deducted %d tokens from user %s", tokensToDeduct, multiExp.UserID)
					}
				} else {
					log.Printf("[ExperimentStatusCron] WARNING: Status is COMPLETED but no completion data available for experiment %d (RunPod ID: %s)", i, exp.RunpodID)
					continue
				}
			case "IN_PROGRESS":
				log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) still in progress", i, exp.RunpodID)
				continue
			case "IN_QUEUE":
				log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) is queued", i, exp.RunpodID)
				continue
			case "FAILED":
				log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) failed", i, exp.RunpodID)
				if exp.RetryCount < MaxRetries {
					log.Printf("[ExperimentStatusCron] Retrying experiment %d (RunPod ID: %s) - attempt %d/%d", i, exp.RunpodID, exp.RetryCount+1, MaxRetries)

					// Resubmit the experiment to RunPod
					requestBody := models.InnocentInputParams{
						S3InputBucketName:    global.S3_INPUT_BUCKET_NAME,
						S3InputFilePath:      fmt.Sprintf("innocent/%s%s", exp.FileID, exp.FileExtension),
						S3OutputBucketName:   global.S3_OUTPUT_BUCKET_NAME,
						S3OutputMaskFilePath: fmt.Sprintf("innocent/%s.png", exp.FileID),
						S3OutputResultsPath:  fmt.Sprintf("innocent/%s.json", exp.FileID),
						S3OutputTablePath:    fmt.Sprintf("innocent/%s.xlsx", exp.FileID),
						NRays:                32,
						MicronsPerPixel:      exp.MicronsPerPixel,
					}

					response, err := models.InnocentMakeRequest(requestBody)
					if err != nil {
						log.Printf("[ExperimentStatusCron] Error resubmitting experiment %d to RunPod: %v", i, err)
						multiExp.Experiments[i].Status = experiments.ExperimentFailed
						continue
					}

					multiExp.Experiments[i].Status = experiments.ExperimentInProgress
					multiExp.Experiments[i].RunpodID = response.ID
					multiExp.Experiments[i].RetryCount++
				} else {
					log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) failed after %d retries", i, exp.RunpodID, MaxRetries)
					multiExp.Experiments[i].Status = experiments.ExperimentFailed
				}
			default:
				log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) has unknown status: %s", i, exp.RunpodID, status.Status)
				if exp.RetryCount < MaxRetries {
					log.Printf("[ExperimentStatusCron] Retrying experiment %d (RunPod ID: %s) - attempt %d/%d", i, exp.RunpodID, exp.RetryCount+1, MaxRetries)

					// Resubmit the experiment to RunPod
					requestBody := models.InnocentInputParams{
						S3InputBucketName:    global.S3_INPUT_BUCKET_NAME,
						S3InputFilePath:      fmt.Sprintf("innocent/%s%s", exp.FileID, exp.FileExtension),
						S3OutputBucketName:   global.S3_OUTPUT_BUCKET_NAME,
						S3OutputMaskFilePath: fmt.Sprintf("innocent/%s.png", exp.FileID),
						S3OutputResultsPath:  fmt.Sprintf("innocent/%s.json", exp.FileID),
						S3OutputTablePath:    fmt.Sprintf("innocent/%s.xlsx", exp.FileID),
						NRays:                32,
						MicronsPerPixel:      exp.MicronsPerPixel,
					}

					response, err := models.InnocentMakeRequest(requestBody)
					if err != nil {
						log.Printf("[ExperimentStatusCron] Error resubmitting experiment %d to RunPod: %v", i, err)
						multiExp.Experiments[i].Status = experiments.ExperimentFailed
						continue
					}

					multiExp.Experiments[i].Status = experiments.ExperimentInProgress
					multiExp.Experiments[i].RunpodID = response.ID
					multiExp.Experiments[i].RetryCount++
				} else {
					log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) failed after %d retries", i, exp.RunpodID, MaxRetries)
					multiExp.Experiments[i].Status = experiments.ExperimentFailed
				}
			}
		}

		// Check if all experiments are completed and generate download URL if needed
		allCompleted := true
		for _, exp := range multiExp.Experiments {
			if exp.Status != experiments.ExperimentCompleted {
				allCompleted = false
				break
			}
		}

		if allCompleted && multiExp.DownloadURL == "" {
			log.Printf("[ExperimentStatusCron] All experiments completed for group %s, generating download URL", multiExp.ID)
			downloadURL, err := experiments.GenerateDownloadLink(multiExp.ID)
			if err != nil {
				log.Printf("[ExperimentStatusCron] Error generating download URL for group %s: %v", multiExp.ID, err)
			} else {
				multiExp.DownloadURL = downloadURL
				log.Printf("[ExperimentStatusCron] Generated download URL for group %s", multiExp.ID)
			}
		}

		// Update the experiment in MongoDB
		_, err = collection.UpdateOne(
			context.Background(),
			bson.M{"_id": multiExp.ID},
			bson.M{"$set": bson.M{
				"experiments": multiExp.Experiments,
				"downloadUrl": multiExp.DownloadURL,
			}},
		)
		if err != nil {
			log.Printf("[ExperimentStatusCron] Error updating experiment group %s in MongoDB: %v", multiExp.ID, err)
			continue
		}
		log.Printf("[ExperimentStatusCron] Successfully updated experiment group %s in MongoDB", multiExp.ID)
	}

	log.Println("[ExperimentStatusCron] Completed experiment status update cycle")
	return nil
}

// StartExperimentStatusCron starts the cron job that updates experiment statuses
func StartExperimentStatusCron() {
	log.Println("[ExperimentStatusCron] Starting experiment status cron job")
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			log.Println("[ExperimentStatusCron] Starting new update cycle")
			if err := UpdateExperimentStatuses(); err != nil {
				log.Printf("[ExperimentStatusCron] Error in update cycle: %v", err)
			}
		}
	}()
}
