package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeX1RequiresNetworkApproval(t *testing.T) {
	c := config(t)
	c.Knowledge = KnowledgeConfig{Mode: "octobus_native_x1"}
	_, err := NewEngine(c)
	requireCode(t, err, "NETWORK_NOT_APPROVED")
}

// This is an offline subprocess double, not evidence of gRPC/OctoBus compatibility.
func nativeTestEngine(t *testing.T, response any) (*Engine, string) {
	t.Helper()
	dir := t.TempDir()
	b, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "response.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n/bin/cat > \"${0%/*}/request.json\"\nprintf 'call\\n' >> \"${0%/*}/calls\"\n/bin/cat \"${0%/*}/response.json\"\n"
	path := filepath.Join(dir, "grpcurl-double")
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CAP_GRPC_TARGET", "127.0.0.1:7411")
	t.Setenv("CAP_TOKEN", "SYNTHETIC_NATIVE_SANDBOX_TOKEN")
	c := config(t)
	c.NetworkApproved = true
	c.Knowledge = KnowledgeConfig{Mode: "octobus_native_x1", Capset: "enterprise/compliance-readonly", Instance: "compliance-kb-synth"}
	e, err := NewEngine(c)
	if err != nil {
		t.Fatal(err)
	}
	e.knowledge.(*nativeKnowledge).executable = path
	return e, dir
}

func TestNativeX1ResultAndReplay(t *testing.T) {
	e, dir := nativeTestEngine(t, map[string]any{"corpus": fixture(t)})
	r := request()
	out, replay, err := e.Query(context.Background(), r)
	if err != nil || replay || out.Status != "REFERENCE_ONLY" || !out.HumanReviewRequired || out.EntailmentVerified {
		t.Fatalf("unexpected native result: result=%v replay=%v err=%v", out, replay, err)
	}
	again, replay, err := e.Query(context.Background(), r)
	if err != nil || !replay || again.InputDigest != out.InputDigest {
		t.Fatalf("completed native request was not replayed: replay=%v err=%v", replay, err)
	}
	calls, err := os.ReadFile(filepath.Join(dir, "calls"))
	if err != nil || string(calls) != "call\n" {
		t.Fatalf("native subprocess was repeated: %q %v", calls, err)
	}
}

func TestNativeX1OversizedOutputDoesNotRetry(t *testing.T) {
	e, _ := nativeTestEngine(t, map[string]string{"padding": strings.Repeat("x", 4*1024*1024+1)})
	_, _, err := e.Query(context.Background(), request())
	requireCode(t, err, "UPSTREAM_RESPONSE_TOO_LARGE")
	_, _, err = e.Query(context.Background(), request())
	requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
}

func TestNativeX1RejectsUnsafeResponses(t *testing.T) {
	nonSynthetic := fixture(t)
	nonSynthetic.Synthetic = false
	for _, tc := range []struct {
		name string
		body any
		code string
	}{
		{"credential", map[string]any{"corpus": fixture(t), "echo": "SYNTHETIC_NATIVE_SANDBOX_TOKEN"}, "CREDENTIAL_REFLECTION_BLOCKED"},
		{"truncated", map[string]any{"corpus": fixture(t), "truncated": true}, "RETRIEVAL_TRUNCATED"},
		{"non-synthetic", map[string]any{"corpus": nonSynthetic}, "NATIVE_X1_SYNTHETIC_ONLY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := nativeTestEngine(t, tc.body)
			out, _, err := e.Query(context.Background(), request())
			requireCode(t, err, tc.code)
			if out != nil || strings.Contains(err.Error(), "SYNTHETIC_NATIVE_SANDBOX_TOKEN") {
				t.Fatal("unsafe upstream material escaped the native adapter")
			}
		})
	}
}

func TestNativeX1DoesNotAcceptDirectCredentials(t *testing.T) {
	c := config(t)
	c.NetworkApproved = true
	c.Knowledge = KnowledgeConfig{Mode: "octobus_native_x1", Capset: "enterprise/compliance-readonly", Instance: "compliance-kb-synth", TokenEnv: "UPSTREAM_TOKEN"}
	_, err := NewEngine(c)
	requireCode(t, err, "NATIVE_CONFIG_INVALID")
}
