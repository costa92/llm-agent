package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/costa92/llm-agent"
	ragcore "github.com/costa92/llm-agent-rag/rag"
)

// AsTool wraps a compatibility RAGSystem as an agents.Tool.
func AsTool(r *RAGSystem) agents.Tool {
	return agents.NewFuncTool(
		"rag",
		"Retrieval-Augmented Generation. Actions: add_text, search, ask, remove, stats.",
		ragToolSchema(),
		ragToolHandler(r),
	)
}

func ragToolSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action":      {"type": "string", "enum": ["add_text","search","ask","remove","stats"]},
			"text":        {"type": "string"},
			"query":       {"type": "string"},
			"question":    {"type": "string"},
			"id":          {"type": "string"},
			"top_k":       {"type": "integer"},
			"namespace":   {"type": "string"},
			"enable_mqe":  {"type": "boolean"},
			"enable_hyde": {"type": "boolean"},
			"mqe_count":   {"type": "integer"},
			"metadata":    {"type": "object"}
		},
		"required": ["action"]
	}`)
}

type ragToolArgs struct {
	Action       string         `json:"action"`
	Text         string         `json:"text"`
	Query        string         `json:"query"`
	Question     string         `json:"question"`
	ID           string         `json:"id"`
	TopK         int            `json:"top_k"`
	Namespace    string         `json:"namespace"`
	EnableMQE    bool           `json:"enable_mqe"`
	EnableHyDE   bool           `json:"enable_hyde"`
	MQECount     int            `json:"mqe_count"`
	EnableRerank bool           `json:"enable_rerank"`
	Metadata     map[string]any `json:"metadata"`
}

func ragToolHandler(r *RAGSystem) agents.ExecuteFunc {
	return func(ctx context.Context, raw json.RawMessage) (string, error) {
		var p ragToolArgs
		if err := json.Unmarshal(raw, &p); err != nil {
			return "", fmt.Errorf("rag: bad args: %w", err)
		}
		switch p.Action {
		case "add_text":
			if p.Text == "" {
				return "", errors.New("rag: text required for add_text")
			}
			md := copyMeta(p.Metadata)
			if p.Namespace != "" {
				md[namespaceMetadataKey] = p.Namespace
			}
			ids, err := r.AddText(ctx, p.Text, md)
			if err != nil {
				return "", err
			}
			b, _ := json.Marshal(map[string]any{"ids": ids, "count": len(ids)})
			return string(b), nil
		case "search":
			if p.Query == "" {
				return "", ErrEmptyQuery
			}
			hits, err := r.searchWithNamespace(ctx, p.Query, p.TopK, p.Namespace, SearchOptions{
				EnableMQE:       p.EnableMQE,
				EnableHyDE:      p.EnableHyDE,
				MQECount:        p.MQECount,
				EnableRerank:    p.EnableRerank,
				Filters:         copyMeta(p.Metadata),
				SecurityFilters: nil,
			})
			if err != nil {
				return "", err
			}
			b, _ := json.Marshal(hits)
			return string(b), nil
		case "ask":
			if p.Question == "" {
				return "", errors.New("rag: question required for ask")
			}
			answer, err := r.core.Ask(ctx, p.Question, ragcore.AskOptions{
				Search: ragcore.SearchOptions{
					TopK:            max(p.TopK, 5),
					Namespace:       p.Namespace,
					Filters:         copyMeta(p.Metadata),
					SecurityFilters: nil,
					EnableMQE:       p.EnableMQE,
					EnableHyDE:      p.EnableHyDE,
					MQECount:        p.MQECount,
					EnableRerank:    p.EnableRerank,
				},
				Metadata: copyMeta(p.Metadata),
			})
			if err != nil {
				return "", err
			}
			b, _ := json.Marshal(map[string]any{
				"answer":      answer.Text,
				"citations":   answer.Citations,
				"diagnostics": answer.Diagnostics,
				"trace":       answer.Trace,
			})
			return string(b), nil
		case "remove":
			if p.ID == "" {
				return "", errors.New("rag: id required for remove")
			}
			if err := r.Remove(ctx, p.ID); err != nil {
				return "", err
			}
			return `{"removed":true}`, nil
		case "stats":
			b, _ := json.Marshal(r.Stats())
			return string(b), nil
		case "":
			return "", errors.New("rag: action is required")
		default:
			return "", fmt.Errorf("rag: unknown action %q", p.Action)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
