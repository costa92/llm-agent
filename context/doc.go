// Package context implements GSSC (Gather→Select→Structure→Compress)
// context engineering for LLM agents.
//
// The Builder accepts BuildInput (system prompt + history + memory hits
// + RAG hits + custom packets) and returns a budget-respecting prompt.
//
// # GSSC pipeline
//
//   - Gather:    every source becomes []Packet, each tagged with Source
//                + Timestamp + TokenCount.
//   - Select:    score = recency × Wᵣ + relevance × Wₗ; drop below
//                MinRelevance; greedy fill until (1-ReserveRatio)*MaxTokens.
//                System packets always survive.
//   - Structure: render kept packets into 5 fixed sections —
//                [Role & Policies] / [Task] / [Evidence] / [Context] / [History]
//   - Compress:  if still over budget, optionally LLM-summarize (when
//                EnableCompress + WithLLM); otherwise hard-truncate with
//                a trailing "[truncated]" marker.
//
// # Plug points
//
//   - WithTokenCounter: swap SimpleCounter for tiktoken-go etc.
//   - WithEmbedder:     swap Jaccard for cosine via rag.Embedder.
//   - WithLLM:          enable LLM-backed Compress phase.
//
// # Naming caveat
//
// This package is named `context` and lives at
// pkg/llm/agents/context. It does NOT replace stdlib `context`.
// When importing both, alias one of them:
//
//	import (
//	    "context"
//
//	    aictx "github.com/costa92/llm-agent/context"
//	)
//
// # Portability
//
// context inherits the agents/pkg/llm portability contract — no
// internal/*, no project pkg/*, no business vocabulary.
package context
