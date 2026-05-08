// Package context implements GSSC (Gather→Select→Structure→Compress)
// context engineering: a pipeline that turns multi-source data into a
// budget-respecting LLM prompt.
//
// NOTE on naming: this package is named `context` and does NOT replace
// stdlib context. Disambiguate with an import alias when both are used:
//
//	import (
//	    "context"
//
//	    aictx "github.com/costa92/llm-agent/context"
//	)
//
// See doc.go for the full pipeline overview.
package context

import "time"

// Source identifies the origin of a Packet, used for grouping during
// the Structure phase.
type Source string

const (
	SourceSystem       Source = "system"
	SourceMemory       Source = "memory"
	SourceRAG          Source = "rag"
	SourceConversation Source = "conversation"
	SourceCustom       Source = "custom"
)

// Packet is the unit the GSSC pipeline operates on. TokenCount is
// filled by Gather; Relevance by Select.
type Packet struct {
	Content    string
	Source     Source
	Timestamp  time.Time
	TokenCount int
	Relevance  float64 // ∈ [0, 1]
	Metadata   map[string]any
}

// Config tunes the GSSC pipeline. Zero-values get sensible defaults
// applied by Builder.
type Config struct {
	MaxTokens       int     // budget cap; default 3000
	ReserveRatio    float64 // fraction reserved for system + meta; default 0.2
	MinRelevance    float64 // packets below this are dropped before Select; default 0.1
	RecencyWeight   float64 // weight on time-decay in Select score; default 0.3
	RelevanceWeight float64 // weight on similarity in Select score; default 0.7
	EnableCompress  bool    // enable Structure-phase truncation/summarization; default true
}

// applyDefaults fills zero fields. Returns a copy so callers' Config
// stays untouched.
func (c Config) applyDefaults() Config {
	out := c
	if out.MaxTokens <= 0 {
		out.MaxTokens = 3000
	}
	if out.ReserveRatio <= 0 {
		out.ReserveRatio = 0.2
	}
	if out.MinRelevance < 0 {
		out.MinRelevance = 0
	}
	if out.MinRelevance == 0 && c.MinRelevance == 0 {
		out.MinRelevance = 0.1
	}
	if out.RecencyWeight <= 0 {
		out.RecencyWeight = 0.3
	}
	if out.RelevanceWeight <= 0 {
		out.RelevanceWeight = 0.7
	}
	// EnableCompress defaults to true when not explicitly disabled by zero
	// — but bool zero value collapses with "not set"; we treat false as
	// "user wants off" by leaving it.
	return out
}
