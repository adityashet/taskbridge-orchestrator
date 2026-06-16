package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"taskbridge/internal/executor"
	"taskbridge/internal/model"
	"time"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "TaskBridge server URL")
	agentID := flag.String("id", "agent-dev-1", "agent identifier")
	capabilities := flag.String("capabilities", "http_check,tcp_check,file_exists", "comma-separated job capabilities")
	pollInterval := flag.Duration("poll-interval", 3*time.Second, "job polling interval")
	flag.Parse()

	fmt.Println("TaskBridge agent started...")
	fmt.Println("server:", *serverURL)
	fmt.Println("agent_id:", *agentID)
	fmt.Println("capabilities:", *capabilities)
	fmt.Println("poll_interval:", *pollInterval)

	// Split comma-separated capabilities into a slice of model.JobType
	capsRaw := strings.Split(*capabilities, ",")
	var caps []model.JobType
	for _, c := range capsRaw {
		caps = append(caps, model.JobType(strings.TrimSpace(c)))
	}

	// 1. Register agent with server
	registerAgent(*serverURL, *agentID, caps)

	// 2. Start periodic heartbeat in the background (Goroutine)
	go startHeartbeatLoop(*serverURL, *agentID)

	// 3. Main loop: Poll server for next job
	startPollingLoop(*serverURL, *agentID, caps, *pollInterval)
}

// registerAgent sends a POST request to register this agent with the server
func registerAgent(serverURL, agentID string, capabilities []model.JobType) {
	url := fmt.Sprintf("%s/agents/register", serverURL)

	agentData := model.Agent{
		ID:           agentID,
		Capabilities: capabilities,
	}

	jsonData, err := json.Marshal(agentData)
	if err != nil {
		log.Printf("[Error] Failed to marshal registration data: %v", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[Error] Could not connect to server for registration: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("[Success] Agent successfully registered with central server.")
	} else {
		log.Printf("[Warning] Registration returned status code: %d", resp.StatusCode)
	}
}

// startHeartbeatLoop sends a heartbeat request every 4 seconds in the background
func startHeartbeatLoop(serverURL, agentID string) {
	// We need to implement POST /agents/{agentId}/heartbeat on the server later,
	// but let's set up the agent loop to fire it now.
	ticker := time.NewTicker(4 * time.Second)
	for range ticker.C {
		url := fmt.Sprintf("%s/agents/%s/heartbeat", serverURL, agentID)
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			log.Printf("[Heartbeat Error] Server unreachable: %v", err)
			continue
		}
		resp.Body.Close()
	}
}

// startPollingLoop continuously asks the server for jobs
func startPollingLoop(serverURL, agentID string, capabilities []model.JobType, interval time.Duration) {
	ticker := time.NewTicker(interval)

	// Struct payload matching what the server expect for AssignNextJob
	type NextJobRequest struct {
		Capabilities []model.JobType `json:"capabilities"`
	}

	for range ticker.C {
		url := fmt.Sprintf("%s/agents/%s/next-job", serverURL, agentID)

		reqPayload := NextJobRequest{Capabilities: capabilities}
		jsonData, _ := json.Marshal(reqPayload)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("[Poll Error] Server unreachable: %v", err)
			continue
		}

		// If server returns 204 or a non-OK status, it means no pending jobs are available
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var job model.Job
		err = json.NewDecoder(resp.Body).Decode(&job)
		resp.Body.Close()

		if err != nil {
			log.Printf("[Error] Failed to decode incoming job: %v", err)
			continue
		}

		log.Printf("[Job Received] Found job ID: %s (%s). Processing...", job.ID, job.Type)

		// Placeholder for Milestone 2 executor steps
		executeJobPlaceholder(serverURL, job)
	}
}

func executeJobPlaceholder(serverURL string, job model.Job) {
	// 1. Initialize the executor engine registry and register our structural workers
	reg := executor.NewRegistry()
	reg.Register(&executor.HTTPCheckExecutor{})
	reg.Register(&executor.TCPCheckExecutor{})
	reg.Register(&executor.FileExistsExecutor{})

	exec, exists := reg.Get(job.Type)
	if !exists {
		log.Printf("[Warning] No compatible local executor found for type: %s", job.Type)
		return
	}

	// 2. Set up the requested execution timeout limits safely
	timeout := 10 * time.Second
	if job.TimeoutSeconds > 0 {
		timeout = time.Duration(job.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 3. Run the task using the matching executor module
	res := exec.Execute(ctx, job)

	// 4. Construct the response payload to report logs and results back to the server
	// 4. Construct the response payload to report logs and results back to the server
	reportUrl := fmt.Sprintf("%s/jobs/%s/result", serverURL, job.ID)

	type ReportPayload struct {
		Status model.JobStatus `json:"status"` // Aligned to model.JobStatus
		Logs   []string        `json:"logs"`
		Result map[string]any  `json:"result,omitempty"`
		Error  string          `json:"error,omitempty"`
	}

	payload := ReportPayload{
		Status: res.Status,
		Logs:   res.Logs,
		Result: res.Result,
		Error:  res.Error,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Error] Failed to marshal job results: %v", err)
		return
	}

	resp, err := http.Post(reportUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[Error] Failed to post job results back to server: %v", err)
		return
	}
	resp.Body.Close()

	log.Printf("[Job Finished] Status: %s reported for Job: %s", res.Status, job.ID)
}
