package experiments

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var MultiExperimentCollection = "multi_experiments"

type MultiExperiment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	Name        string             `bson:"name" json:"name"`
	Experiments []Experiment       `bson:"experiments,omitempty" json:"experiments,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	DownloadURL string             `bson:"downloadUrl,omitempty" json:"downloadUrl,omitempty"`
}

type Experiment struct {
	FileID              string           `bson:"fileId,omitempty" json:"fileId,omitempty"`
	FileExtension       string           `bson:"fileExtension,omitempty" json:"fileExtension,omitempty"`
	RunpodID            string           `bson:"runpodID,omitempty" json:"runpodID,omitempty"`
	ExecutionTimeMillis int              `bson:"executionTimeMillis,omitempty" json:"executionTimeMillis,omitempty"`
	Status              ExperimentStatus `bson:"status,omitempty" json:"status,omitempty"`
	RetryCount          int              `bson:"retryCount,omitempty" json:"retryCount,omitempty"`
	MicronsPerPixel     float64          `bson:"micronsPerPixel,omitempty" json:"micronsPerPixel,omitempty"`
}

type ExperimentStatus string

const (
	ExperimentInQueue    ExperimentStatus = "IN_QUEUE"
	ExperimentInProgress ExperimentStatus = "IN_PROGRESS"
	ExperimentCompleted  ExperimentStatus = "COMPLETED"
	ExperimentFailed     ExperimentStatus = "FAILED"
)
