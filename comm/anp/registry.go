// Package anp is a minimal Agent Network Protocol implementation:
// in-memory service discovery + load-aware selection. NO DID, no
// signing, no key verification — those are the defining ANP features
// in real deployments and out of scope for Phase 5 minimal.
//
// What's covered:
//
//   - Service / Registry types
//   - Register / Discover / GetBest with caller-supplied scoreFn
//   - Default score = 1 / (1 + load), so lower-load services win
//   - AsAgentTool: expose registry operations as a single agents.Tool
//
// # Portability
//
// anp inherits the agents/pkg/llm portability contract.
package anp

import (
	"errors"
	"sort"
	"sync"
)

// Service is one registered network endpoint. Endpoints is a list so
// a service can publish multiple URLs (HTTP + gRPC + …).
type Service struct {
	ID          string
	Type        string         // "compute" / "search" / ...
	Endpoints   []string
	Metadata    map[string]any // load / region / cpu_cores / ...
	HealthScore float64        // [0,1]; caller-set, not auto-probed
}

// Registry holds Services in memory under RWMutex.
type Registry struct {
	mu       sync.RWMutex
	services map[string]*Service
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry { return &Registry{services: make(map[string]*Service)} }

// Sentinel errors.
var (
	ErrServiceIDRequired = errors.New("anp: service.ID is required")
	ErrNoMatch           = errors.New("anp: no service matches")
)

// Register adds (or replaces) a service by ID. Returns
// ErrServiceIDRequired if ID is empty.
func (r *Registry) Register(s *Service) error {
	if s == nil || s.ID == "" {
		return ErrServiceIDRequired
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	r.services[s.ID] = &cp
	return nil
}

// Unregister removes a service by ID. No-op if absent.
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, id)
}

// Discover returns all registered services whose Type matches.
// Pass "" to get every service.
func (r *Registry) Discover(serviceType string) []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Service, 0, len(r.services))
	for _, s := range r.services {
		if serviceType == "" || s.Type == serviceType {
			cp := *s
			out = append(out, &cp)
		}
	}
	// Stable order so callers see deterministic results.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// GetBest returns the highest-scoring service of the given type. If
// scoreFn is nil, the default 1/(1+load) is used. Returns ErrNoMatch
// when no service of that type is registered.
func (r *Registry) GetBest(serviceType string, scoreFn func(*Service) float64) (*Service, error) {
	if scoreFn == nil {
		scoreFn = defaultScoreFn
	}
	candidates := r.Discover(serviceType)
	if len(candidates) == 0 {
		return nil, ErrNoMatch
	}
	bestIdx := 0
	bestScore := scoreFn(candidates[0])
	for i := 1; i < len(candidates); i++ {
		s := scoreFn(candidates[i])
		if s > bestScore {
			bestScore = s
			bestIdx = i
		}
	}
	return candidates[bestIdx], nil
}

// defaultScoreFn favors low load + high health. Looks up "load" in
// Metadata (float64 or int); missing → 0 load. Score = HealthScore /
// (1 + load).
func defaultScoreFn(s *Service) float64 {
	load := 0.0
	if v, ok := s.Metadata["load"]; ok {
		switch n := v.(type) {
		case float64:
			load = n
		case int:
			load = float64(n)
		case int64:
			load = float64(n)
		}
	}
	health := s.HealthScore
	if health <= 0 {
		health = 1 // assume healthy when not annotated
	}
	return health / (1 + load)
}

// Stats returns the count of registered services.
func (r *Registry) Stats() (count int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.services)
}
