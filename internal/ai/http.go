package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Request tuning for single-shot JSON tasks. Extraction and narration
// want low temperature and bounded output. Shapes follow the Meta Model
// API docs: https://dev.meta.ai/docs/protocols/responses and
// https://dev.meta.ai/docs/structured-output
const (
	modelTemperature = 0.2
	maxOutputTokens  = 8192
)

// NewHTTPModel builds a Meta Model API client for the Responses API.
// endpoint is the full URL, e.g. https://api.meta.ai/v1/responses.
func NewHTTPModel(endpoint, model, token string) (LLM, error) {
	if endpoint == "" || model == "" {
		return nil, fmt.Errorf("ai: set FLATMATE_MODEL_ENDPOINT and FLATMATE_MODEL_NAME")
	}
	return &httpModel{endpoint: endpoint, model: model, token: token,
		client: &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			// Headers wait out almost the whole budget: some gateways
			// buffer reasoning models until generation completes.
			ResponseHeaderTimeout: 55 * time.Second,
		}}}, nil
}

// NewHTTPModelFromEnv builds a model client from the environment.
func NewHTTPModelFromEnv() (LLM, error) {
	return NewHTTPModel(os.Getenv("FLATMATE_MODEL_ENDPOINT"), os.Getenv("FLATMATE_MODEL_NAME"), os.Getenv("FLATMATE_MODEL_TOKEN"))
}

type httpModel struct {
	endpoint string
	model    string
	token    string
	client   *http.Client
}

// CompleteRaw runs one stateless Responses call with server-enforced
// JSON schema output. store=false keeps prompts out of server history.
func (m *httpModel) CompleteRaw(ctx context.Context, prompt, schema string) (json.RawMessage, error) {
	var schemaDoc any
	if err := json.Unmarshal([]byte(schema), &schemaDoc); err != nil {
		return nil, fmt.Errorf("ai: bad schema: %w", err)
	}
	enforceStrict(schemaDoc)

	body, _ := json.Marshal(map[string]any{
		"model":             m.model,
		"input":             prompt,
		"temperature":       modelTemperature,
		"max_output_tokens": maxOutputTokens,
		"store":             false,
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "FlatmateResult",
				"schema": schemaDoc,
				"strict": true,
			},
		},
	})
	req, err := http.NewRequestWithContext(ctx, "POST", m.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snip, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return nil, fmt.Errorf("ai: status %s: %s", resp.Status, strings.TrimSpace(string(snip)))
	}
	var envelope responsesEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return envelope.text()
}

// responsesEnvelope is the subset of a Responses API reply we read:
// completion status plus assistant message text.
type responsesEnvelope struct {
	Status string `json:"status"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

// text concatenates the assistant's output_text blocks. Anything else —
// an incomplete response, a reasoning-only turn, an empty reply —
// reports an error so the caller falls back to deterministic output.
func (e responsesEnvelope) text() (json.RawMessage, error) {
	if e.Status != "" && e.Status != "completed" {
		return nil, fmt.Errorf("ai: response status %q", e.Status)
	}
	var b strings.Builder
	for _, item := range e.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" {
				b.WriteString(c.Text)
			}
		}
	}
	if b.Len() == 0 {
		return nil, fmt.Errorf("ai: no output_text in response")
	}
	return json.RawMessage(b.String()), nil
}

// enforceStrict makes a decoded JSON schema satisfy strict-mode rules:
// every object gets additionalProperties:false and all properties required.
func enforceStrict(node any) {
	n, ok := node.(map[string]any)
	if !ok {
		return
	}
	if props, ok := n["properties"].(map[string]any); ok {
		n["additionalProperties"] = false
		req := make([]string, 0, len(props))
		for k, p := range props {
			req = append(req, k)
			enforceStrict(p)
		}
		sort.Strings(req)
		n["required"] = req
	} else if t, _ := n["type"].(string); t == "object" {
		n["additionalProperties"] = false
	}
	if items, ok := n["items"]; ok {
		enforceStrict(items)
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if arr, ok := n[key].([]any); ok {
			for _, s := range arr {
				enforceStrict(s)
			}
		}
	}
	for _, key := range []string{"$defs", "definitions"} {
		if defs, ok := n[key].(map[string]any); ok {
			for _, d := range defs {
				enforceStrict(d)
			}
		}
	}
}
