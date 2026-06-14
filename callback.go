package agents

import "context"

type callbackAgent struct {
	inner Agent
	cb    Callback
}

// WrapAgent returns an Agent decorator that mirrors unified RunEvents to cb.
// It does not change the wrapped Agent's StepEvent stream or errors.
func WrapAgent(inner Agent, cb Callback) Agent {
	if inner == nil {
		return nil
	}
	return &callbackAgent{inner: inner, cb: cb}
}

// ObserveAgent is an alias for WrapAgent for call sites that prefer observer
// wording.
func ObserveAgent(inner Agent, cb Callback) Agent {
	return WrapAgent(inner, cb)
}

func (a *callbackAgent) Name() string { return a.inner.Name() }

func (a *callbackAgent) Run(ctx context.Context, input string) (Result, error) {
	res, err := a.inner.Run(ctx, input)
	if err != nil {
		EmitRunEvent(ctx, a.cb, RunEvent{Kind: RunEventAgentError, AgentName: a.Name(), Err: err})
		return res, err
	}
	for _, step := range res.Trace {
		EmitRunEvent(ctx, a.cb, RunEventFromStepEvent(a.Name(), StepEvent{Step: step}))
	}
	EmitRunEvent(ctx, a.cb, RunEventFromStepEvent(a.Name(), StepEvent{Done: true, Final: &res}))
	return res, nil
}

func (a *callbackAgent) RunStream(ctx context.Context, input string) (<-chan StepEvent, error) {
	ch, err := a.inner.RunStream(ctx, input)
	if err != nil {
		EmitRunEvent(ctx, a.cb, RunEvent{Kind: RunEventAgentError, AgentName: a.Name(), Err: err})
		return nil, err
	}
	out := make(chan StepEvent, 16)
	go func() {
		defer close(out)
		for ev := range ch {
			EmitRunEvent(ctx, a.cb, RunEventFromStepEvent(a.Name(), ev))
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

var _ Agent = (*callbackAgent)(nil)
