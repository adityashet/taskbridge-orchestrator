package executor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"taskbridge/internal/model"
)

// Result is returned after executing a job.
type Result struct {
	Status model.JobStatus `json:"status"`
	Logs   []string        `json:"logs"`
	Result map[string]any  `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Executor executes a single job type.
type Executor interface {
	Type() model.JobType
	Execute(ctx context.Context, job model.Job) Result
}

// Registry maps job types to executors.
type Registry struct {
	executors map[model.JobType]Executor
}

func NewRegistry() *Registry {
	return &Registry{executors: map[model.JobType]Executor{}}
}

func (r *Registry) Register(ex Executor) {
	r.executors[ex.Type()] = ex
}

func (r *Registry) Get(t model.JobType) (Executor, bool) {
	ex, ok := r.executors[t]
	return ex, ok
}

// --- 1. HTTP CHECK EXECUTOR ---
type HTTPCheckExecutor struct{}

func (e *HTTPCheckExecutor) Type() model.JobType { return "http_check" }

func (e *HTTPCheckExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := []string{"Starting HTTP check..."}

	url, _ := job.Payload["url"].(string)
	expectedStatusRaw, _ := job.Payload["expected_status"]

	// Safely convert expected status from float64 (JSON default) or int
	expectedStatus := 200
	if val, ok := expectedStatusRaw.(float64); ok {
		expectedStatus = int(val)
	} else if val, ok := expectedStatusRaw.(int); ok {
		expectedStatus = val
	}

	if url == "" {
		return Result{Status: "FAILED", Logs: append(logs, "Error: missing target URL"), Error: "missing url parameter"}
	}

	logs = append(logs, fmt.Sprintf("Sending GET request to %s, expecting status %d", url, expectedStatus))

	// Create an HTTP request bound to the task context (handles timeouts automatically)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{Status: "FAILED", Logs: append(logs, fmt.Sprintf("Error creating request: %v", err)), Error: err.Error()}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Status: "FAILED", Logs: append(logs, fmt.Sprintf("Request execution failed: %v", err)), Error: err.Error()}
	}
	defer resp.Body.Close()

	logs = append(logs, fmt.Sprintf("Received status code: %d", resp.StatusCode))

	if resp.StatusCode == expectedStatus {
		return Result{Status: "SUCCESS", Logs: append(logs, "HTTP check successful match!"), Result: map[string]any{"status_code": resp.StatusCode}}
	}

	errMsg := fmt.Sprintf("Status code mismatch: expected %d, got %d", expectedStatus, resp.StatusCode)
	return Result{Status: "FAILED", Logs: append(logs, errMsg), Error: errMsg}
}

// --- 2. TCP CHECK EXECUTOR ---
type TCPCheckExecutor struct{}

func (e *TCPCheckExecutor) Type() model.JobType { return "tcp_check" }

func (e *TCPCheckExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := []string{"Starting TCP reachability check..."}
	address, _ := job.Payload["address"].(string)

	if address == "" {
		return Result{Status: "FAILED", Logs: append(logs, "Error: missing target address"), Error: "missing address parameter"}
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return Result{Status: "FAILED", Logs: append(logs, fmt.Sprintf("Failed to connect to %s: %v", address, err)), Error: err.Error()}
	}
	conn.Close()

	return Result{Status: "SUCCESS", Logs: append(logs, fmt.Sprintf("Successfully established TCP link to %s", address))}
}

// --- 3. FILE EXISTS EXECUTOR ---
type FileExistsExecutor struct{}

func (e *FileExistsExecutor) Type() model.JobType { return "file_exists" }

func (e *FileExistsExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := []string{"Starting file path validation..."}
	path, _ := job.Payload["path"].(string)

	if path == "" {
		return Result{Status: "FAILED", Logs: append(logs, "Error: missing target file path"), Error: "missing path parameter"}
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return Result{Status: "FAILED", Logs: append(logs, fmt.Sprintf("File path does not exist: %s", path)), Error: "file not found"}
	} else if err != nil {
		return Result{Status: "FAILED", Logs: append(logs, fmt.Sprintf("Error checking file: %v", err)), Error: err.Error()}
	}

	return Result{Status: "SUCCESS", Logs: append(logs, fmt.Sprintf("Target file verified. Size: %d bytes", info.Size()))}
}
