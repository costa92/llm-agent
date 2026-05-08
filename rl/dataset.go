// Package rl implements the EVALUATION layer of Agentic Reinforcement
// Learning. Training (SFT / GRPO / PPO / LoRA / gradient backprop)
// requires Python's TRL/transformers/torch stack and is INTENTIONALLY
// out of scope. Use TrainerProxy to bridge to a real Python backend.
//
// What this package gives you:
//
//   - Dataset / Sample loaders (JSONL + GSM8K parser)
//   - Trajectory / Episode types
//   - Reward interface + 4 built-ins (Accuracy, LengthPenalty, StepBonus, Composite)
//   - Evaluator with bounded concurrency + ctx timeout + Pass@K sampling
//   - Pure metric functions: ComputeAccuracy / ComputePassAtK /
//     ComputeFormatCorrectness / ComputeNumericError
//   - TrainerProxy interface + UnsupportedTrainer no-op
//
// # Portability
//
// rl inherits the agents/pkg/llm portability contract — no internal/*,
// no project pkg/*, no business vocabulary.
package rl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"strings"
)

// Sample is a single eval input. GroundTruth carries the expected
// answer (extracted by the dataset's parser); Metadata is opaque.
type Sample struct {
	ID          string
	Prompt      string
	GroundTruth string
	Metadata    map[string]any
}

// Dataset is a stream of Samples. Iter returns a Go 1.23+ iter.Seq
// so callers can lazy-iterate large files without loading them all.
type Dataset interface {
	Iter(ctx context.Context) iter.Seq[Sample]
	Len() int
	Name() string
}

// LineParser converts one JSONL line into a Sample.
type LineParser func(line []byte) (Sample, error)

// JSONLDataset loads samples line-by-line from a .jsonl file. Each
// line is fed to a per-dataset LineParser.
type JSONLDataset struct {
	name   string
	path   string
	parser LineParser
	count  int // computed on Len() lazily
}

// NewJSONLDataset opens path and validates it exists. Sample parsing
// is deferred to Iter (so missing/corrupt lines surface during eval,
// not at construction).
func NewJSONLDataset(name, path string, parser LineParser) (*JSONLDataset, error) {
	if name == "" {
		return nil, errors.New("rl: dataset name required")
	}
	if parser == nil {
		return nil, errors.New("rl: dataset parser required")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("rl: dataset %q: %w", name, err)
	}
	ds := &JSONLDataset{name: name, path: path, parser: parser, count: -1}
	return ds, nil
}

// Name implements Dataset.
func (d *JSONLDataset) Name() string { return d.name }

// Len implements Dataset. Counts non-blank lines on first call (cached
// thereafter). Returns -1 if the file becomes unreadable mid-stream.
func (d *JSONLDataset) Len() int {
	if d.count >= 0 {
		return d.count
	}
	f, err := os.Open(d.path)
	if err != nil {
		return -1
	}
	defer f.Close()
	count := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			count++
		}
	}
	d.count = count
	return count
}

// Iter implements Dataset. ctx-cancellable. Lines that fail to parse
// are silently dropped — pre-flight your dataset if you need strict
// validation.
func (d *JSONLDataset) Iter(ctx context.Context) iter.Seq[Sample] {
	return func(yield func(Sample) bool) {
		f, err := os.Open(d.path)
		if err != nil {
			return
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := sc.Bytes()
			if len(strings.TrimSpace(string(line))) == 0 {
				continue
			}
			s, err := d.parser(append([]byte(nil), line...))
			if err != nil {
				continue
			}
			if !yield(s) {
				return
			}
		}
	}
}

// ParseGSM8K parses the official GSM8K JSONL format:
//
//	{"question": "...", "answer": "Reasoning... #### 42"}
//
// GroundTruth is the literal numeric string after "#### ". Trailing
// whitespace / commas removed.
func ParseGSM8K(line []byte) (Sample, error) {
	var raw struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
		ID       string `json:"id"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return Sample{}, fmt.Errorf("gsm8k: parse: %w", err)
	}
	if raw.Question == "" || raw.Answer == "" {
		return Sample{}, errors.New("gsm8k: missing question or answer")
	}
	idx := strings.Index(raw.Answer, "####")
	if idx < 0 {
		return Sample{}, errors.New("gsm8k: missing '####' marker in answer")
	}
	gt := strings.TrimSpace(raw.Answer[idx+4:])
	gt = strings.TrimSuffix(gt, ".")
	gt = strings.ReplaceAll(gt, ",", "")
	id := raw.ID
	if id == "" {
		// Fallback: hash of question
		id = fmt.Sprintf("gsm8k_%x", simpleHash(raw.Question))
	}
	return Sample{
		ID:          id,
		Prompt:      raw.Question,
		GroundTruth: gt,
		Metadata:    map[string]any{"reasoning": raw.Answer[:idx]},
	}, nil
}

// simpleHash is a non-cryptographic FNV-32 over s.
func simpleHash(s string) uint32 {
	const offset, prime = 2166136261, 16777619
	h := uint32(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	return h
}
