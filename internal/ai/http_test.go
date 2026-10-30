package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const probeSchema = `{"type":"object","properties":{"answers":{"type":"array"}},"required":["answers"]}`

// roundTripStub captures the outgoing request and replays a canned reply.
// No sockets, so this runs even where localhost listeners are forbidden.
type roundTripStub struct {
	t       *testing.T
	gotReq  *http.Request
	gotBody map[string]any
	status  int
	reply   string
	called  bool
}

func (s *roundTripStub) RoundTrip(r *http.Request) (*http.Response, error) {
	s.called = true
	s.gotReq = r
	if err := json.NewDecoder(r.Body).Decode(&s.gotBody); err != nil {
		s.t.Errorf("decode request: %v", err)
	}
	return &http.Response{
		Status:        "200 OK",
		StatusCode:    s.status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          io.NopCloser(strings.NewReader(s.reply)),
		ContentLength: int64(len(s.reply)),
		Header:        make(http.Header),
		Request:       r,
	}, nil
}

func stubbedModel(t *testing.T, status int, reply string) (*httpModel, *roundTripStub) {
	t.Helper()
	stub := &roundTripStub{t: t, status: status, reply: reply}
	m, err := NewHTTPModel("https://api.meta.ai/v1/responses", "muse-wonder-1.0", "tok")
	if err != nil {
		t.Fatal(err)
	}
	hm := m.(*httpModel)
	hm.client.Transport = stub
	return hm, stub
}

func TestCompleteRawResponses(t *testing.T) {
	m, stub := stubbedModel(t, 200, `{"id":"resp_1","status":"completed","output":[
		{"type":"reasoning","id":"rs_1"},
		{"type":"message","role":"assistant","status":"completed","content":[
			{"type":"output_text","text":"{\"answers\":[]}","annotations":[]}
		]}]}`)
	raw, err := m.CompleteRaw(context.Background(), "extract this", probeSchema)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if !stub.called {
		t.Fatal("transport not called")
	}
	if string(raw) != `{"answers":[]}` {
		t.Errorf("raw = %s", raw)
	}
	if stub.gotReq.Method != "POST" {
		t.Errorf("method = %s", stub.gotReq.Method)
	}
	if h := stub.gotReq.Header; h.Get("Authorization") != "Bearer tok" ||
		h.Get("Content-Type") != "application/json" {
		t.Errorf("headers = %v", h)
	}
	b := stub.gotBody
	if b["model"] != "muse-wonder-1.0" || b["input"] != "extract this" {
		t.Errorf("model/input = %v", b)
	}
	if b["temperature"] != 0.2 || b["store"] != false {
		t.Errorf("tuning = %v", b)
	}
	if b["max_output_tokens"] != float64(8192) {
		t.Errorf("max_output_tokens = %v", b["max_output_tokens"])
	}
	format, _ := b["text"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" || format["name"] != "FlatmateResult" || format["strict"] != true {
		t.Fatalf("format = %v", b["text"])
	}
	schemaDoc, _ := format["schema"].(map[string]any)
	if schemaDoc["type"] != "object" {
		t.Fatalf("schema not embedded as object: %v", format)
	}
	if schemaDoc["additionalProperties"] != false {
		t.Errorf("schema not strictified: %v", schemaDoc)
	}
	req, _ := schemaDoc["required"].([]any)
	if len(req) != 1 || req[0] != "answers" {
		t.Errorf("required = %v", schemaDoc["required"])
	}
}

func TestEnforceStrict(t *testing.T) {
	var doc any
	if err := json.Unmarshal([]byte(`{"type":"object","properties":{
		"a":{"type":"string"},
		"o":{"type":"object","properties":{"b":{"type":"integer"}}},
		"arr":{"type":"array","items":{"type":"object","properties":{"c":{"type":"string"}}}}
	}}`), &doc); err != nil {
		t.Fatal(err)
	}
	enforceStrict(doc)
	top := doc.(map[string]any)
	if top["additionalProperties"] != false {
		t.Errorf("top additionalProperties = %v", top["additionalProperties"])
	}
	if req, _ := top["required"].([]string); len(req) != 3 {
		t.Errorf("top required = %v", top["required"])
	}
	nested := top["properties"].(map[string]any)["o"].(map[string]any)
	if nested["additionalProperties"] != false {
		t.Errorf("nested additionalProperties = %v", nested["additionalProperties"])
	}
	if req, _ := nested["required"].([]string); len(req) != 1 || req[0] != "b" {
		t.Errorf("nested required = %v", nested["required"])
	}
	items := top["properties"].(map[string]any)["arr"].(map[string]any)["items"].(map[string]any)
	if items["additionalProperties"] != false {
		t.Errorf("items additionalProperties = %v", items["additionalProperties"])
	}
}

func TestCompleteRawErrors(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"http error":     {500, `{"error":"boom"}`},
		"bad json":       {200, `not json`},
		"incomplete":     {200, `{"status":"incomplete","output":[]}`},
		"empty output":   {200, `{"status":"completed","output":[]}`},
		"reasoning only": {200, `{"status":"completed","output":[{"type":"reasoning","id":"rs_1"}]}`},
		"no output_text": {200, `{"status":"completed","output":[{"type":"message","content":[{"type":"other"}]}]}`},
		"empty content":  {200, `{"status":"completed","output":[{"type":"message","content":[]}]}`},
	}
	for name, tc := range cases {
		m, _ := stubbedModel(t, tc.status, tc.body)
		if _, err := m.CompleteRaw(context.Background(), "p", probeSchema); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	m, stub := stubbedModel(t, 200, `{}`)
	if _, err := m.CompleteRaw(context.Background(), "p", "{bad"); err == nil {
		t.Error("bad schema: want error")
	}
	if stub.called {
		t.Error("bad schema: must fail before sending")
	}
	if _, err := NewHTTPModel("", "", ""); err == nil {
		t.Error("empty config: want error")
	}
}

func TestCompleteRawMultiBlock(t *testing.T) {
	m, _ := stubbedModel(t, 200, `{"status":"completed","output":[{"type":"message","content":[
		{"type":"output_text","text":"{\"a\":"},
		{"type":"output_text","text":"1}"}
	]}]}`)
	raw, err := m.CompleteRaw(context.Background(), "p", probeSchema)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != `{"a":1}` {
		t.Errorf("joined = %s", raw)
	}
}

func TestCompleteJSONRoundTrip(t *testing.T) {
	m, _ := stubbedModel(t, 200, `{"status":"completed","output":[{"type":"message",
		"content":[{"type":"output_text","text":"{\"answers\":[{\"dimension\":\"sleep\",\"value\":\"late\"}]}"}]}]}`)
	out, err := CompleteJSON(context.Background(), m, "p", extractSchema, validateExtract)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Answers) != 1 || out.Answers[0].Dimension != "sleep" {
		t.Errorf("parsed = %+v", out)
	}
}
