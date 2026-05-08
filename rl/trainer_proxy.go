package rl

import (
	"context"
	"errors"

	"github.com/costa92/llm-agent/llm"
)

// TrainerProxy is the interface a Python-bridged trainer would
// implement. The default implementation (UnsupportedTrainer) returns
// ErrTrainingNotSupported on every call.
//
// To wire a real trainer:
//  1. Stand up a Python TRL service (sft/grpo).
//  2. Write a bridge type satisfying TrainerProxy that calls into
//     it (HTTP, gRPC, subprocess).
//  3. Construct via your bridge instead of UnsupportedTrainer.
type TrainerProxy interface {
	Train(ctx context.Context, cfg TrainConfig) (TrainResult, error)
	LoadModel(path string) (llm.Client, error)
}

// TrainConfig captures everything needed by a backend to train.
// Hyperparams stays open-ended to avoid Go-side enumeration of every
// algorithm-specific knob.
type TrainConfig struct {
	Algorithm   string  // "sft" | "grpo" | "ppo" — backend-defined
	BaseModel   string  // HF model ID or local path
	Dataset     Dataset // streamed by the backend
	Reward      Reward  // GRPO/PPO reward; ignored by SFT
	OutputDir   string
	Hyperparams map[string]any
}

// TrainResult is the post-training summary.
type TrainResult struct {
	OutputDir string
	FinalLoss float64
	Metrics   Metrics
}

// ErrTrainingNotSupported is returned by UnsupportedTrainer's Train
// and LoadModel. Callers should detect this with errors.Is and
// either skip training or instantiate a real backend.
var ErrTrainingNotSupported = errors.New("rl: training requires external Python backend (TRL); use a TrainerProxy bridge")

// UnsupportedTrainer is the default no-op implementation of TrainerProxy.
type UnsupportedTrainer struct{}

// NewUnsupportedTrainer returns the no-op trainer.
func NewUnsupportedTrainer() *UnsupportedTrainer { return &UnsupportedTrainer{} }

// Train always returns ErrTrainingNotSupported.
func (UnsupportedTrainer) Train(_ context.Context, _ TrainConfig) (TrainResult, error) {
	return TrainResult{}, ErrTrainingNotSupported
}

// LoadModel always returns ErrTrainingNotSupported.
func (UnsupportedTrainer) LoadModel(_ string) (llm.Client, error) {
	return nil, ErrTrainingNotSupported
}
