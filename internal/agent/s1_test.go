package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestS1InvalidOutputNeverCompletes(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"missing_claims", `{"choices":[{"finish_reason":"stop","message":{"content":"{}"}}]}`, "MODEL_OUTPUT_INVALID"},
		{"null_claims", `{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":null}"}}]}`, "MODEL_OUTPUT_INVALID"},
		{"legacy_function", `{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[]}","function_call":{"name":"run"}}}]}`, "MODEL_OUTPUT_INCOMPLETE_OR_TOOL_CALL"},
		{"duplicate_envelope_key", `{"choices":[],"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[]}"}}]}`, "MODEL_RESPONSE_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				fmt.Fprint(w, syntheticEnvelope(tc.body))
			}))
			defer server.Close()
			t.Setenv("TEST_LLM_TOKEN", "SYNTHETIC_MODEL_TOKEN_123456")
			c := config(t)
			c.NetworkApproved = true
			c.Generation = GenerationConfig{Mode: "llm", Endpoint: server.URL, Model: generationModel, TokenEnv: "TEST_LLM_TOKEN"}
			e, err := NewEngine(c)
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := e.Query(context.Background(), request())
			requireCode(t, err, tc.code)
			if out != nil {
				t.Fatal("invalid model output became a result")
			}
			_, _, err = e.Query(context.Background(), request())
			requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
			if calls.Load() != 1 {
				t.Fatal("retried invalid output")
			}
		})
	}
}

func TestS1TokenUsageIsRecordedWithoutQuotaProof(t *testing.T) {
	for _, tc := range []struct{ name, usage, want string }{
		{"reported", `{"prompt_tokens":120,"completion_tokens":30,"total_tokens":150,"prompt_tokens_details":{"cached_tokens":70},"completion_tokens_details":{"reasoning_tokens":20}}`, `{"status":"reported","inputTokens":120,"outputTokens":30,"totalTokens":150,"cachedInputTokens":70,"reasoningTokens":20}`},
		{"provider_total", `{"prompt_tokens":26,"completion_tokens":168,"total_tokens":498,"prompt_tokens_details":{"cached_tokens":4},"completion_tokens_details":{"reasoning_tokens":304}}`, `{"status":"reported","inputTokens":26,"outputTokens":168,"totalTokens":498,"cachedInputTokens":4,"reasoningTokens":304}`},
		{"reported_zero", `{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}`, `{"status":"reported","inputTokens":0,"outputTokens":0,"totalTokens":0}`},
		{"partial", `{"prompt_tokens":120,"completion_tokens":30}`, `{"status":"partial","inputTokens":120,"outputTokens":30}`},
		{"missing", `null`, `{"status":"unknown"}`},
		{"malformed", `{"prompt_tokens":"120"}`, `{"status":"unknown"}`},
		{"negative", `{"prompt_tokens":-1,"completion_tokens":30,"total_tokens":29}`, `{"status":"unknown"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				fmt.Fprintf(w, `{"model":"grok-4.6","object":"chat.completion","usage":%s,"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[]}","reasoning_content":"PRIVATE_REASONING_SENTINEL"}}]}`, tc.usage)
			}))
			defer server.Close()
			t.Setenv("TEST_LLM_TOKEN", "SYNTHETIC_MODEL_TOKEN_123456")
			c := config(t)
			c.NetworkApproved = true
			c.Generation = GenerationConfig{Mode: "llm", Endpoint: server.URL, Model: "grok-4.6", TokenEnv: "TEST_LLM_TOKEN"}
			e, err := NewEngine(c)
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := e.Query(context.Background(), request())
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(out)
			var result map[string]json.RawMessage
			var got, want map[string]any
			if json.Unmarshal(b, &result) != nil || json.Unmarshal(result["tokenUsage"], &got) != nil || json.Unmarshal([]byte(tc.want), &want) != nil {
				t.Fatal("token usage missing from query result")
			}
			want["provider"], want["model"] = "baizhi-chat", "grok-4.6"
			if hashJSON(got) != hashJSON(want) {
				t.Fatalf("usage = %s; want %s", result["tokenUsage"], hashJSON(want))
			}
			replay, replayed, err := e.Query(context.Background(), request())
			if err != nil || !replayed || hashJSON(replay) != hashJSON(out) || calls.Load() != 1 {
				t.Fatal("replay changed usage or sent another model request")
			}
			saved, err := os.ReadFile(filepath.Join(c.StoreDir, request().RequestID+".json"))
			if err != nil || strings.Contains(string(saved), "PRIVATE_REASONING_SENTINEL") || strings.Contains(string(saved), "SYNTHETIC_MODEL_TOKEN_123456") {
				t.Fatal("receipt unavailable or retained private model data")
			}
		})
	}
}

// Fabricated response counters exercise recording, not real model tokenization.
func syntheticEnvelope(choices string) string {
	return `{"model":"grok-4.6","object":"chat.completion","usage":{"prompt_tokens":120,"completion_tokens":30,"total_tokens":150,"prompt_tokens_details":{"cached_tokens":70},"completion_tokens_details":{"reasoning_tokens":20}},` + choices[1:]
}
