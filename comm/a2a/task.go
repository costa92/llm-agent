// Package a2a is a simplified Agent-to-Agent protocol: HTTP server
// exposing skills + an async task state machine, plus a client that
// invokes remote skills (POST → poll → artifact).
//
// This is NOT wire-compatible with Google's a2a-sdk — the schema is
// custom-and-tiny so the concept fits in ≤500 LOC. Use comm.HTTPTransport
// with a real a2a server and write your own translator if you need
// interop.
//
// # Portability
//
// a2a inherits the agents/pkg/llm portability contract.
package a2a

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// SkillDescriptor describes one skill the server exposes.
type SkillDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// TaskState is the state machine for one server-side task.
type TaskState string

const (
	TaskPending   TaskState = "pending"
	TaskRunning   TaskState = "running"
	TaskCompleted TaskState = "completed"
	TaskFailed    TaskState = "failed"
)

// Task is one server-side task entry. Artifact is the produced result
// (only populated when State == TaskCompleted).
type Task struct {
	ID         string          `json:"id"`
	Skill      string          `json:"skill"`
	Input      json.RawMessage `json:"input"`
	State      TaskState       `json:"state"`
	Artifact   json.RawMessage `json:"artifact,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Sentinel errors.
var (
	ErrSkillNotFound = errors.New("a2a: skill not found")
	ErrTaskNotFound  = errors.New("a2a: task not found")
)

// taskStore is an in-memory map[id]*Task with RWMutex. Safe for
// concurrent use across HTTP handlers + the worker goroutine.
type taskStore struct {
	mu    sync.RWMutex
	items map[string]*Task
}

func newTaskStore() *taskStore { return &taskStore{items: make(map[string]*Task)} }

func (s *taskStore) put(t *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[t.ID] = t
}

func (s *taskStore) get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.items[id]
	if !ok {
		return nil, false
	}
	cp := *t
	return &cp, true
}

func (s *taskStore) update(id string, fn func(*Task)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return false
	}
	fn(t)
	t.UpdatedAt = time.Now().UTC()
	return true
}
