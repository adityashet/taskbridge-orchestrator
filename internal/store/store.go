package store

import (
	"errors"
	"sync"
	"taskbridge/internal/model"
	"time"
)

// Store defines the required persistence operations.
type Store interface {
	CreateJob(job model.Job) (model.Job, error)
	ListJobs() ([]model.Job, error)
	GetJob(jobID string) (model.Job, bool, error)
	CancelJob(jobID string) error

	RegisterAgent(agent model.Agent) (model.Agent, error)
	Heartbeat(agentID string) error
	ListAgents() ([]model.Agent, error)

	AssignNextJob(agentID string, capabilities []model.JobType) (model.Job, bool, error)
	CompleteJob(jobID string, status model.JobStatus, logs []string, result map[string]any, errMsg string) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	jobs   map[string]model.Job
	agents map[string]model.Agent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs:   make(map[string]model.Job),
		agents: make(map[string]model.Agent),
	}
}

func (ms *MemoryStore) CreateJob(job model.Job) (model.Job, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	job.CreatedAt = time.Now()
	job.Status = model.JobPending
	ms.jobs[job.ID] = job
	return job, nil
}

func (ms *MemoryStore) ListJobs() ([]model.Job, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var allJobs []model.Job
	for _, job := range ms.jobs {
		allJobs = append(allJobs, job)
	}
	return allJobs, nil
}

func (ms *MemoryStore) GetJob(jobID string) (model.Job, bool, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	job, exists := ms.jobs[jobID]
	return job, exists, nil
}

func (ms *MemoryStore) CancelJob(jobID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	job, exists := ms.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.Status = model.JobCanceled
	ms.jobs[jobID] = job
	return nil
}

func (ms *MemoryStore) RegisterAgent(agent model.Agent) (model.Agent, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	agent.LastSeen = time.Now()
	agent.Status = "ONLINE"
	ms.agents[agent.ID] = agent
	return agent, nil
}

func (ms *MemoryStore) Heartbeat(agentID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	agent, exists := ms.agents[agentID]
	if !exists {
		return errors.New("agent not found")
	}
	agent.LastSeen = time.Now()
	ms.agents[agentID] = agent
	return nil
}

func (ms *MemoryStore) ListAgents() ([]model.Agent, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var allAgents []model.Agent
	for _, agent := range ms.agents {
		allAgents = append(allAgents, agent)
	}
	return allAgents, nil
}

func (ms *MemoryStore) AssignNextJob(agentID string, capabilities []model.JobType) (model.Job, bool, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, job := range ms.jobs {
		if job.Status == model.JobPending {
			for _, cap := range capabilities {
				if cap == job.Type {
					now := time.Now()
					job.Status = model.JobRunning
					job.AssignedAgentID = agentID
					job.StartedAt = &now
					job.AttemptCount++

					ms.jobs[job.ID] = job
					return job, true, nil
				}
			}
		}
	}
	return model.Job{}, false, nil
}

func (ms *MemoryStore) CompleteJob(jobID string, status model.JobStatus, logs []string, result map[string]any, errMsg string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	job, exists := ms.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}

	now := time.Now()
	job.Status = status
	job.FinishedAt = &now
	job.Logs = append(job.Logs, logs...)
	job.Result = result
	job.Error = errMsg

	ms.jobs[jobID] = job
	return nil
}
