package api

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"taskbridge/internal/model"
	"taskbridge/internal/store"
	"time"
)

func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return "job-" + string(b)
}

type Handler struct {
	store store.Store
}

func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Catch-all route parsing to dynamically process parameters safely
	mux.HandleFunc("/", h.CoreMuxRouter)
}

func (h *Handler) CoreMuxRouter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	// Route 1: /jobs
	if len(parts) == 1 && parts[0] == "jobs" {
		h.HandleBaseJobs(w, r)
		return
	}

	// Route 2: /jobs/{id}/result
	if len(parts) == 3 && parts[0] == "jobs" && parts[2] == "result" {
		h.HandleJobResultSubmission(w, r, parts[1])
		return
	}

	// Route 3: /agents/register
	if len(parts) == 2 && parts[0] == "agents" && parts[1] == "register" {
		h.HandleRegisterAgent(w, r)
		return
	}

	// Route 4: /agents/{id}/heartbeat or /agents/{id}/next-job
	if len(parts) == 3 && parts[0] == "agents" {
		h.HandleAgentSubActions(w, r, parts[1], parts[2])
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (h *Handler) HandleBaseJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req model.Job
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Type == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		req.ID = generateID()
		createdJob, _ := h.store.CreateJob(req)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createdJob)
	case http.MethodGet:
		jobs, _ := h.store.ListJobs()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jobs)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) HandleJobResultSubmission(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Status model.JobStatus `json:"status"`
		Logs   []string        `json:"logs"`
		Result map[string]any  `json:"result,omitempty"`
		Error  string          `json:"error,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := h.store.CompleteJob(jobID, body.Status, body.Logs, body.Result, body.Error)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"success"}`))
}

func (h *Handler) HandleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req model.Agent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	registeredAgent, _ := h.store.RegisterAgent(req)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(registeredAgent)
}

func (h *Handler) HandleAgentSubActions(w http.ResponseWriter, r *http.Request, agentID, action string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch action {
	case "heartbeat":
		_ = h.store.Heartbeat(agentID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	case "next-job":
		var body struct {
			Capabilities []model.JobType `json:"capabilities"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		job, found, _ := h.store.AssignNextJob(agentID, body.Capabilities)
		if !found {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(job)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
