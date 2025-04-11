package models

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"api.lnlink.net/src/pkg/global"
)

var POST_URL = "https://api.runpod.ai/v2/d4je3qggs32rgy/run"
var GET_URL = "https://api.runpod.ai/v2/d4je3qggs32rgy/status"
var API_KEY = "r0wnvfq3gxwfzm"

type InnocentInputParams struct {
	S3InputBucketName    string  `json:"s3_input_bucket_name"`
	S3InputFilePath      string  `json:"s3_input_file_path"`
	S3OutputBucketName   string  `json:"s3_output_bucket_name"`
	S3OutputMaskFilePath string  `json:"s3_output_mask_file_path"`
	S3OutputResultsPath  string  `json:"s3_output_results_file_path"`
	S3OutputTablePath    string  `json:"s3_output_table_file_path"`
	NRays                int     `json:"n_rays"`
	MicronsPerPixel      float64 `json:"microns_per_pixel"`
}

type InnocentRequestBody struct {
	Input InnocentInputParams `json:"input"`
}

// Base response fields that are common across all statuses
type InnocentBaseResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Response for IN_QUEUE status
type InnocentQueuedResponse struct {
	InnocentBaseResponse
}

// Response for IN_PROGRESS status
type InnocentInProgressResponse struct {
	InnocentBaseResponse
	DelayTime int64  `json:"delayTime"`
	WorkerID  string `json:"workerId"`
}

// Response for COMPLETED status
type InnocentCompletedResponse struct {
	InnocentBaseResponse
	DelayTime     int64  `json:"delayTime"`
	ExecutionTime int64  `json:"executionTime"`
	WorkerID      string `json:"workerId"`
}

// InnocentResponse represents the union of all possible response types
type InnocentResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	*InnocentQueuedResponse
	*InnocentInProgressResponse
	*InnocentCompletedResponse
}

func InnocentMakeRequest(inputParams InnocentInputParams) (*InnocentResponse, error) {
	requestBody := InnocentRequestBody{
		Input: inputParams,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", POST_URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+global.RUNPOD_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response InnocentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func InnocentGetStatus(id string) (*InnocentResponse, error) {
	req, err := http.NewRequest("GET", GET_URL+"/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+global.RUNPOD_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response InnocentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
