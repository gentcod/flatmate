// Package ai holds the Muse touchpoints. The model never ranks,
// filters, or scores; it only narrates precomputed results.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// LLM produces raw JSON for a prompt. Implementations must honor ctx.
type LLM interface {
	CompleteRaw(ctx context.Context, prompt, schema string) (json.RawMessage, error)
}

// CompleteJSON calls the model, parses, and validates, retrying once.
// Callers fall back to deterministic output on error.
func CompleteJSON[T any](ctx context.Context, llm LLM, prompt, schema string, validate func(T) error) (T, error) {
	var zero T
	var err error
	for range 2 {
		var raw json.RawMessage
		raw, err = llm.CompleteRaw(ctx, prompt, schema)
		if err != nil {
			continue
		}
		var v T
		if jerr := json.Unmarshal(raw, &v); jerr != nil {
			err = jerr
			continue
		}
		if verr := validate(v); verr != nil {
			err = verr
			continue
		}
		return v, nil
	}
	return zero, err
}

type disabledLLM struct{}

// Disabled returns an LLM that always errors, forcing fallback output.
// The server uses this unless a model endpoint is configured.
func Disabled() LLM { return disabledLLM{} }

func (disabledLLM) CompleteRaw(ctx context.Context, prompt, schema string) (json.RawMessage, error) {
	return nil, fmt.Errorf("ai: model not configured")
}
