package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/costa92/llm-agent"
)

// AsTool wraps a Manager as an agents.Tool. The schema accepts an
// `action` discriminator; payload fields are action-specific.
//
// Supported actions: add / search / get / update / remove /
// consolidate / forget / stats.
//
// JSON output shape varies per action; format docs in the spec §6.5.
func AsTool(mgr *Manager) agents.Tool {
	return agents.NewFuncTool(
		"memory",
		"Persistent in-process memory (working / episodic / semantic). Actions: add, search, get, update, remove, consolidate, forget, stats.",
		toolSchema(),
		toolHandler(mgr),
	)
}

func toolSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action":     {"type": "string", "enum": ["add","search","get","update","remove","consolidate","forget","stats"]},
			"kind":       {"type": "string", "enum": ["working","episodic","semantic"]},
			"content":    {"type": "string"},
			"id":         {"type": "string"},
			"query":      {"type": "string"},
			"top_k":      {"type": "integer"},
			"importance": {"type": "number"},
			"tags":       {"type": "array", "items": {"type": "string"}},
			"threshold":  {"type": "number"},
			"strategy":   {"type": "string", "enum": ["importance","age","capacity"]},
			"max_age_seconds": {"type": "integer"},
			"keep":       {"type": "integer"},
			"min_age_seconds": {"type": "integer"}
		},
		"required": ["action"]
	}`)
}

type toolArgs struct {
	Action        string   `json:"action"`
	Kind          string   `json:"kind"`
	Content       string   `json:"content"`
	ID            string   `json:"id"`
	Query         string   `json:"query"`
	TopK          int      `json:"top_k"`
	Importance    float64  `json:"importance"`
	Tags          []string `json:"tags"`
	Threshold     float64  `json:"threshold"`
	Strategy      string   `json:"strategy"`
	MaxAgeSeconds int      `json:"max_age_seconds"`
	Keep          int      `json:"keep"`
	MinAgeSeconds int      `json:"min_age_seconds"`
}

func toolHandler(mgr *Manager) agents.ExecuteFunc {
	return func(ctx context.Context, raw json.RawMessage) (string, error) {
		var p toolArgs
		if err := json.Unmarshal(raw, &p); err != nil {
			return "", fmt.Errorf("memory: bad args: %w", err)
		}
		switch p.Action {
		case "add":
			return doAdd(ctx, mgr, p)
		case "search":
			return doSearch(ctx, mgr, p)
		case "get":
			return doGet(ctx, mgr, p)
		case "update":
			return doUpdate(ctx, mgr, p)
		case "remove":
			return doRemove(ctx, mgr, p)
		case "consolidate":
			return doConsolidate(ctx, mgr, p)
		case "forget":
			return doForget(ctx, mgr, p)
		case "stats":
			return doStats(mgr)
		case "":
			return "", errors.New("memory: action is required")
		default:
			return "", fmt.Errorf("memory: unknown action %q", p.Action)
		}
	}
}

func doAdd(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	if strings.TrimSpace(p.Content) == "" {
		return "", errors.New("memory: content required for add")
	}
	id, err := mgr.Add(ctx, Kind(p.Kind), MemoryItem{
		Content:    p.Content,
		Tags:       p.Tags,
		Importance: p.Importance,
	})
	if err != nil {
		return "", err
	}
	return jsonOut(map[string]string{"id": id})
}

func doSearch(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	if strings.TrimSpace(p.Query) == "" {
		return "", ErrEmptyQuery
	}
	if p.Kind == "" {
		// SearchAll across active memories
		out, err := mgr.SearchAll(ctx, p.Query, p.TopK)
		if err != nil {
			return "", err
		}
		return jsonOut(out)
	}
	res, err := mgr.Search(ctx, Kind(p.Kind), p.Query, p.TopK)
	if err != nil {
		return "", err
	}
	return jsonOut(res)
}

func doGet(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	if p.ID == "" {
		return "", errors.New("memory: id required for get")
	}
	item, err := mgr.Get(ctx, Kind(p.Kind), p.ID)
	if err != nil {
		return "", err
	}
	return jsonOut(item)
}

func doUpdate(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	if p.ID == "" {
		return "", errors.New("memory: id required for update")
	}
	err := mgr.Update(ctx, Kind(p.Kind), p.ID, func(it *MemoryItem) {
		if p.Content != "" {
			it.Content = p.Content
		}
		if p.Tags != nil {
			it.Tags = p.Tags
		}
		if p.Importance > 0 {
			it.Importance = p.Importance
		}
	})
	if err != nil {
		return "", err
	}
	return jsonOut(map[string]string{"updated": p.ID})
}

func doRemove(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	if p.ID == "" {
		return "", errors.New("memory: id required for remove")
	}
	if err := mgr.Remove(ctx, Kind(p.Kind), p.ID); err != nil {
		return "", err
	}
	return jsonOut(map[string]string{"removed": p.ID})
}

func doConsolidate(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	count, err := mgr.Consolidate(ctx, ConsolidateOptions{
		Threshold: p.Threshold,
		MinAge:    time.Duration(p.MinAgeSeconds) * time.Second,
	})
	if err != nil {
		return "", err
	}
	return jsonOut(map[string]int{"consolidated": count})
}

func doForget(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	if p.Kind == "" {
		return "", errors.New("memory: kind required for forget")
	}
	count, err := mgr.Forget(ctx, Kind(p.Kind), ForgetOptions{
		Strategy:  ForgetStrategy(p.Strategy),
		Threshold: p.Threshold,
		MaxAge:    time.Duration(p.MaxAgeSeconds) * time.Second,
		Keep:      p.Keep,
	})
	if err != nil {
		return "", err
	}
	return jsonOut(map[string]int{"forgot": count})
}

func doStats(mgr *Manager) (string, error) {
	return jsonOut(mgr.StatsAll())
}

func jsonOut(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("memory: marshal output: %w", err)
	}
	return string(b), nil
}
