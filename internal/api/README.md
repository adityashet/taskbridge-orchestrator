

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"taskbridge/internal/model"
	"taskbridge/internal/store"

	// If your project uses standard crypto or a custom uuid, you can use a basic random string generator
	// to avoid path dependency errors. Let's use a lightweight manual generator for safety:
	"math/rand"
	"time"
)

// Helper function to generate quick job IDs without external packages
func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return "job-" + string(b)
}

// Handler holds our database store interface reference
type Handler struct {
	store store.Store
}

// NewHandler initializes the API handler wrapper
func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

// RegisterRoutes hooks up the endpoints to a multiplexer
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/jobs", h.HandleJobs)
	mux.HandleFunc("/agents/register", h.HandleRegisterAgent)
}

// HandleJobs routes GET /jobs and POST /jobs cleanly
func (h *Handler) HandleJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		var req model.Job
		// JSON request decoding
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON request body"})
			return
		}

		// Input validation
		if req.Name == "" || req.Type == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing required fields: name and type"})
			return
		}

		// Setup default structural fields
		req.JobID = generateID()
		
		// Calling the Store interface
		createdJob, err := h.store.CreateJob(req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to persist job record"})
			return
		}

		// JSON response encoding
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createdJob)

	case http.MethodGet:
		jobs, err := h.store.ListJobs()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch jobs"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jobs)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method profile not supported"})
	}
}

// HandleRegisterAgent processes POST /agents/register
func (h *Handler) HandleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req model.Agent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.AgentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing agent_id"})
		return
	}

	registeredAgent, err := h.store.RegisterAgent(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not register agent"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(registeredAgent)
}