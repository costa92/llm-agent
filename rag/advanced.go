package rag

import (
	"context"

	ragadvanced "github.com/costa92/llm-agent-rag/advanced"
	raggenerate "github.com/costa92/llm-agent-rag/generate"
)

func (r *RAGSystem) mqeExpand(ctx context.Context, query string, n int) ([]string, error) {
	return ragadvanced.ExpandQuery(ctx, modelAdapter{inner: r.llm}, query, n)
}

func (r *RAGSystem) hydeGenerate(ctx context.Context, query string) (string, error) {
	return ragadvanced.GenerateHypothetical(ctx, modelAdapter{inner: r.llm}, query)
}

var _ raggenerate.Model = modelAdapter{}
