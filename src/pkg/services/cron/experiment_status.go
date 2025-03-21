package cron

import (
	"context"
	"log"
	"math"
	"time"

	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/experiments"
	"api.lnlink.net/src/pkg/models/user"
	"api.lnlink.net/src/pkg/services/models"
	"go.mongodb.org/mongo-driver/bson"
)

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

					// Calculate tokens to deduct (1 token per minute, rounded up)
					executionMinutes := float64(status.InnocentCompletedResponse.ExecutionTime) / 60000 // Convert ms to minutes
					tokensToDeduct := int(math.Ceil(executionMinutes))

					log.Printf("[ExperimentStatusCron] Deducting %d tokens for experiment %d (RunPod ID: %s) - execution time: %.2f minutes",
						tokensToDeduct, i, exp.RunpodID, executionMinutes)

					// Get user and deduct tokens
					userID := multiExp.UserID
					user := user.GetUserByID(userID)
					if user != nil {
						user.AddTokens(-tokensToDeduct)
						log.Printf("[ExperimentStatusCron] Successfully deducted %d tokens from user %s", tokensToDeduct, userID)
					} else {
						log.Printf("[ExperimentStatusCron] WARNING: Could not find user %s to deduct tokens", userID)
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
				multiExp.Experiments[i].Status = experiments.ExperimentFailed
			default:
				log.Printf("[ExperimentStatusCron] Experiment %d (RunPod ID: %s) has unknown status: %s", i, exp.RunpodID, status.Status)
				multiExp.Experiments[i].Status = experiments.ExperimentFailed
			}
		}

		// Update the experiment in MongoDB
		_, err = collection.UpdateOne(
			context.Background(),
			bson.M{"_id": multiExp.ID},
			bson.M{"$set": bson.M{"experiments": multiExp.Experiments}},
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
