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
// consolidate / forget / stats / list / pin / unpin / disable / enable.
//
// JSON output shape varies per action; format docs in the spec §6.5.
func AsTool(mgr *Manager) agents.Tool {
	return agents.NewFuncTool(
		"memory",
		"Persistent in-process memory (working / episodic / semantic). Actions: add, search, get, update, remove, consolidate, forget, stats, list, pin, unpin, disable, enable.",
		toolSchema(),
		toolHandler(mgr),
	)
}

func toolSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action":     {"type": "string", "enum": ["add","search","get","update","remove","consolidate","forget","stats","list","pin","unpin","disable","enable"]},
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
			"min_age_seconds": {"type": "integer"},
			"page_size":  {"type": "integer"},
			"cursor":     {"type": "string"},
			"cursors":    {"type": "object", "additionalProperties": {"type": "string"}},
			"filter": {
				"type": "object",
				"properties": {
					"user_scope":       {"type": "string"},
					"project_scope":    {"type": "string"},
					"session_scope":    {"type": "string"},
					"source":           {"type": "string"},
					"category":         {"type": "string"},
					"tags":             {"type": "array", "items": {"type": "string"}},
					"pinned_only":      {"type": "boolean"},
					"include_disabled": {"type": "boolean"},
					"min_importance":   {"type": "number"}
				}
			}
		},
		"required": ["action"]
	}`)
}

type toolArgs struct {
	Action        string            `json:"action"`
	Kind          string            `json:"kind"`
	Content       string            `json:"content"`
	ID            string            `json:"id"`
	Query         string            `json:"query"`
	TopK          int               `json:"top_k"`
	Importance    float64           `json:"importance"`
	Tags          []string          `json:"tags"`
	Threshold     float64           `json:"threshold"`
	Strategy      string            `json:"strategy"`
	MaxAgeSeconds int               `json:"max_age_seconds"`
	Keep          int               `json:"keep"`
	MinAgeSeconds int               `json:"min_age_seconds"`
	Filter        *toolFilter       `json:"filter,omitempty"`
	PageSize      int               `json:"page_size"`
	Cursor        string            `json:"cursor"`
	Cursors       map[string]string `json:"cursors"`
}

// toolFilter is the JSON wire form of ListFilter. Empty/zero fields
// translate to "no constraint" for that axis. Scope is flattened into
// three string axes so the JSON schema stays flat (no nested object
// for the scope itself).
type toolFilter struct {
	UserScope       string   `json:"user_scope"`
	ProjectScope    string   `json:"project_scope"`
	SessionScope    string   `json:"session_scope"`
	Source          string   `json:"source"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags"`
	PinnedOnly      bool     `json:"pinned_only"`
	IncludeDisabled bool     `json:"include_disabled"`
	MinImportance   float64  `json:"min_importance"`
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
		case "list":
			return doList(ctx, mgr, p)
		case "pin":
			return doPin(ctx, mgr, p, true)
		case "unpin":
			return doPin(ctx, mgr, p, false)
		case "disable":
			return doDisable(ctx, mgr, p, true)
		case "enable":
			return doDisable(ctx, mgr, p, false)
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

func doList(ctx context.Context, mgr *Manager, p toolArgs) (string, error) {
	f := buildListFilter(p.Filter)
	if p.Kind == "" {
		// fan out via ListAll
		cursors := make(map[Kind]string, len(p.Cursors))
		for k, v := range p.Cursors {
			cursors[Kind(k)] = v
		}
		out, err := mgr.ListAll(ctx, f, p.PageSize, cursors)
		if err != nil {
			return "", err
		}
		return jsonOut(out)
	}
	mem, err := mgr.lookup(Kind(p.Kind))
	if err != nil {
		return "", err
	}
	lister, ok := mem.(Lister)
	if !ok {
		return "", fmt.Errorf("memory: kind %s does not support list", p.Kind)
	}
	page, err := lister.List(ctx, f, p.PageSize, p.Cursor)
	if err != nil {
		return "", err
	}
	return jsonOut(page)
}

func doPin(ctx context.Context, mgr *Manager, p toolArgs, pinned bool) (string, error) {
	if p.ID == "" {
		return "", errors.New("memory: id required")
	}
	err := mgr.Update(ctx, Kind(p.Kind), p.ID, func(it *MemoryItem) {
		SetPinned(it, pinned)
	})
	if err != nil {
		return "", err
	}
	return jsonOut(map[string]any{"id": p.ID, "pinned": pinned})
}

func doDisable(ctx context.Context, mgr *Manager, p toolArgs, disabled bool) (string, error) {
	if p.ID == "" {
		return "", errors.New("memory: id required")
	}
	err := mgr.Update(ctx, Kind(p.Kind), p.ID, func(it *MemoryItem) {
		SetDisabled(it, disabled)
	})
	if err != nil {
		return "", err
	}
	return jsonOut(map[string]any{"id": p.ID, "disabled": disabled})
}

func buildListFilter(p *toolFilter) ListFilter {
	if p == nil {
		return ListFilter{}
	}
	return ListFilter{
		Scope:           Scope{User: p.UserScope, Project: p.ProjectScope, Session: p.SessionScope},
		Source:          Source(p.Source),
		Category:        Category(p.Category),
		Tags:            p.Tags,
		PinnedOnly:      p.PinnedOnly,
		IncludeDisabled: p.IncludeDisabled,
		MinImportance:   p.MinImportance,
	}
}

func jsonOut(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("memory: marshal output: %w", err)
	}
	return string(b), nil
}
