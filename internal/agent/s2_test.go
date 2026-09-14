package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Uses the existing native subprocess double and a kernel-mount-table fixture.
// This is not a real mount, agent-compose proxy, OctoBus, or Grok attestation.
func managedTestConfig(t *testing.T, origin string, response any) (Config, []byte, string) {
	t.Helper()
	base, dir := nativeTestEngine(t, response)
	c := base.cfg
	var err error
	c.StoreDir, err = filepath.EvalSymlinks(c.StoreDir)
	if err != nil {
		t.Fatal(err)
	}
	c.Knowledge.Mode = "octobus_native_s2"
	c.Generation = GenerationConfig{Mode: "llm", Model: generationModel, TokenEnv: "OPENAI_API_KEY"}
	c.Managed = &ManagedConfig{DaemonOrigin: origin, Project: "cqa-s2-02", Role: "compliance-query"}
	t.Setenv("OPENAI_BASE_URL", origin+"/api/runtime/sandboxes/"+strings.Repeat("a", 64)+"/llm/openai/v1")
	t.Setenv("OPENAI_API_KEY", "SYNTHETIC_FACADE_TOKEN_A")
	mount := []byte("25 20 0:1 /receipts " + c.StoreDir + " rw,nosuid - ext4 /dev/test rw\n")
	return c, mount, dir
}

func managedTestEngine(t *testing.T, c Config, mount []byte, dir string) *Engine {
	t.Helper()
	e, err := newEngine(c, mount)
	if err != nil {
		t.Fatal(err)
	}
	e.knowledge.(*nativeKnowledge).executable = filepath.Join(dir, "grpcurl-double")
	return e
}

func TestS2ManagedReplayAndStableIdentity(t *testing.T) {
	var received, forwarded atomic.Int32
	corpus, err := LoadCorpus("../../testdata/s1-test-document.json", true)
	if err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded.Add(1)
		var body map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 5 || string(body["model"]) != `"grok-4.6"` {
			t.Error("managed generation changed the minimum model contract")
		}
		fmt.Fprint(w, syntheticEnvelope(strings.ReplaceAll(`{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[{\"text\":\"合成参考草稿\",\"evidenceIds\":[\"demo-mlps-assets\"]}]}"}}]}`, "demo-mlps-assets", corpus.Sources[0].ID)))
	}))
	defer upstream.Close()
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		if r.URL.Path != "/api/runtime/sandboxes/"+strings.Repeat("a", 64)+"/llm/openai/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer SYNTHETIC_FACADE_TOKEN_A" {
			t.Error("wrong sandbox route or credential")
		}
		body, _ := io.ReadAll(r.Body)
		if bytes.Contains(body, []byte("private-case")) || bytes.Contains(body, []byte("SYNTHETIC_FACADE")) {
			t.Error("private correlation or credential entered model data")
		}
		out, err := postJSON(r.Context(), newHTTPClient(3), upstream.URL, "SYNTHETIC_DAEMON_UPSTREAM_TOKEN", body, false)
		if err != nil {
			t.Error(err)
			w.WriteHeader(502)
			return
		}
		w.Write(out)
	}))
	defer proxy.Close()
	c, mount, dir := managedTestConfig(t, proxy.URL, map[string]any{"corpus": corpus})
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	e := managedTestEngine(t, c, mount, dir)
	input, err := os.ReadFile("../../examples/s1-test.json")
	var r Request
	if err != nil || StrictJSON(input, &r) != nil {
		t.Fatal("invalid fixed single-document request", err)
	}
	r.RequestID = "s2-managed-normal"
	r.CaseID, r.InvocationID, r.ContextDigest = "private-case", "private-invocation", strings.Repeat("b", 64)
	one, replay, err := e.Query(context.Background(), r)
	if err != nil || replay || one.Status != "DRAFT_READY" || one.DatasetID != corpus.DatasetID || len(one.Citations) != 1 || one.Citations[0].SHA256 != corpus.Sources[0].SHA256 || one.DataMode != "synthetic_demo" || !one.HumanReviewRequired || one.EntailmentVerified || one.TokenUsage.Status != "reported" {
		t.Fatalf("managed draft failed: %v", err)
	}
	path := filepath.Join(c.StoreDir, r.RequestID+".json")
	before, _ := os.ReadFile(path)
	// New sandbox URL, capability target port and both credentials; same trust/config/build.
	t.Setenv("OPENAI_BASE_URL", c.Managed.DaemonOrigin+"/api/runtime/sandboxes/"+strings.Repeat("c", 64)+"/llm/openai/v1")
	t.Setenv("OPENAI_API_KEY", "SYNTHETIC_FACADE_TOKEN_B")
	t.Setenv("CAP_TOKEN", "SYNTHETIC_CAP_TOKEN_B")
	t.Setenv("CAP_GRPC_TARGET", "127.0.0.1:7412")
	next := managedTestEngine(t, c, mount, dir)
	two, replay, err := next.Query(context.Background(), r)
	after, _ := os.ReadFile(path)
	a, _ := json.MarshalIndent(one, "", "  ")
	b, _ := json.MarshalIndent(two, "", "  ")
	calls, _ := os.ReadFile(filepath.Join(dir, "calls"))
	if err != nil || !replay || !bytes.Equal(a, b) || !bytes.Equal(before, after) || string(calls) != "call\n" || received.Load() != 1 || forwarded.Load() != 1 {
		t.Fatal("completed replay changed bytes or sent Search/proxy/upstream again", err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.TimeoutSeconds++ },
		func(c *Config) { c.Knowledge.Capset = "enterprise/other" },
		func(c *Config) { c.Knowledge.Instance = "another-instance" },
		func(c *Config) { c.Managed.Project = "another-project" },
		func(c *Config) { c.Managed.Role = "another-role" },
	} {
		changed := c
		managed := *c.Managed
		changed.Managed = &managed
		change(&changed)
		_, _, err := managedTestEngine(t, changed, mount, dir).Query(context.Background(), r)
		requireCode(t, err, "IDEMPOTENCY_CONFLICT")
	}
	for _, change := range []func(*Request){func(r *Request) { r.Question += " changed" }, func(r *Request) { r.InvocationID = "different-invocation" }} {
		changed := r
		change(&changed)
		_, _, err := next.Query(context.Background(), changed)
		requireCode(t, err, "IDEMPOTENCY_CONFLICT")
	}
	if received.Load() != 1 || forwarded.Load() != 1 {
		t.Fatal("identity conflicts sent model traffic")
	}
	// The checksum protects accidental corruption, not malicious administrator edits.
	var saved receipt
	if StrictJSON(before, &saved) != nil {
		t.Fatal("invalid completed receipt")
	}
	saved.Result.CaseID = "wrong-case"
	corrupt, _ := json.Marshal(saved)
	if err := os.WriteFile(path, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err = next.Query(context.Background(), r)
	requireCode(t, err, "RECEIPT_CORRUPT")
	calls, _ = os.ReadFile(filepath.Join(dir, "calls"))
	if string(calls) != "call\n" || received.Load() != 1 || forwarded.Load() != 1 {
		t.Fatal("corrupt receipt sent traffic")
	}
}

func TestS2ManagedAdmissionAndMountRefusal(t *testing.T) {
	var sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { sends.Add(1) }))
	defer server.Close()
	c, mount, dir := managedTestConfig(t, server.URL, map[string]any{"corpus": fixture(t)})
	for _, tc := range []struct {
		name   string
		change func(*Config)
		code   string
	}{
		{"network", func(c *Config) { c.NetworkApproved = false }, "NETWORK_NOT_APPROVED"},
		{"old-x1", func(c *Config) { c.Managed = nil; c.Knowledge.Mode = "octobus_native_x1" }, "NATIVE_X1_SYNTHETIC_ONLY"},
		{"formal", func(c *Config) { c.AllowSynthetic = false }, "MANAGED_TEST_ONLY"},
		{"model", func(c *Config) { c.Generation.Model = "other-model" }, "MODEL_CONFIG_INVALID"},
		{"direct", func(c *Config) { c.Generation.Endpoint = generationEndpoint }, "MANAGED_CONFIG_INVALID"},
		{"upstream-key", func(c *Config) { c.Generation.TokenEnv = "BAIZHI_API_KEY" }, "MANAGED_CONFIG_INVALID"},
		{"public-http", func(c *Config) { c.Managed.DaemonOrigin = "http://attacker.invalid" }, "MANAGED_DAEMON_INVALID"},
		{"missing-root", func(c *Config) { c.StoreDir += "/absent" }, "PERSISTENT_STORE_REQUIRED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := c
			managed := *c.Managed
			changed.Managed = &managed
			tc.change(&changed)
			_, err := newEngine(changed, mount)
			requireCode(t, err, tc.code)
		})
	}
	for _, table := range [][]byte{nil, []byte("not a kernel mount table"), bytes.ReplaceAll(mount, []byte(" rw"), []byte(" ro")), bytes.Replace(mount, []byte(c.StoreDir), []byte(filepath.Dir(c.StoreDir)), 1), bytes.Replace(mount, []byte("ext4"), []byte("tmpfs"), 1)} {
		_, err := newEngine(c, table)
		requireCode(t, err, "PERSISTENT_STORE_REQUIRED")
	}
	if err := os.Chmod(c.StoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := newEngine(c, mount)
	requireCode(t, err, "STORE_PERMISSIONS_UNSAFE")
	if err := os.Chmod(c.StoreDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		generationEndpoint,
		server.URL + "/api/runtime/sandboxes/" + strings.Repeat("a", 64) + "/llm/openai/v1?token=secret",
		server.URL + "/api/runtime/sandboxes/" + strings.Repeat("a", 64) + "/llm/openai/v1/responses",
		"http://127.0.0.1:1/api/runtime/sandboxes/" + strings.Repeat("a", 64) + "/llm/openai/v1",
	} {
		t.Setenv("OPENAI_BASE_URL", raw)
		_, err := newEngine(c, mount)
		requireCode(t, err, "MANAGED_MODEL_ROUTE_INVALID")
	}
	t.Setenv("OPENAI_BASE_URL", server.URL+"/api/runtime/sandboxes/"+strings.Repeat("a", 64)+"/llm/openai/v1")
	t.Setenv("OPENAI_API_KEY", "")
	_, err = newEngine(c, mount)
	requireCode(t, err, "MODEL_CREDENTIAL_MISSING")
	if _, err := os.Stat(c.StoreDir + "/absent"); !os.IsNotExist(err) {
		t.Fatal("created missing persistent root")
	}
	if _, err := os.Stat(filepath.Join(dir, "calls")); !os.IsNotExist(err) || sends.Load() != 0 {
		t.Fatal("preflight sent traffic")
	}
}

func TestS2ManagedSourceGates(t *testing.T) {
	var sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { sends.Add(1) }))
	defer server.Close()
	for _, tc := range []struct {
		name, question, status, code string
		change                       func(*Corpus)
	}{
		{"no-evidence", "审计日志应保留多久", "INSUFFICIENT_EVIDENCE", "", func(*Corpus) {}},
		{"unreviewed", "资产清单", "NEEDS_REVIEW", "", func(c *Corpus) { c.Sources[0].CuratorReviewed = false }},
		{"unknown-currency", "资产清单", "NEEDS_REVIEW", "", func(c *Corpus) { c.Sources[0].ValidityCheckedAt = "" }},
		{"stale", "资产清单", "NEEDS_REVIEW", "", func(c *Corpus) { c.Sources[0].ValidityCheckedAt = "2026-01-01" }},
		{"conflict", "重叠版本测试", "NEEDS_REVIEW", "", func(*Corpus) {}},
		{"tampered", "资产清单", "", "SOURCE_DIGEST_MISMATCH", func(c *Corpus) { c.Sources[0].Text += "changed" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			corpus := fixture(t)
			tc.change(&corpus)
			c, mount, dir := managedTestConfig(t, server.URL, map[string]any{"corpus": corpus})
			e := managedTestEngine(t, c, mount, dir)
			r := request()
			r.Question = tc.question
			out, _, err := e.Query(context.Background(), r)
			if tc.code != "" {
				requireCode(t, err, tc.code)
			} else if err != nil || out.Status != tc.status || len(out.Claims) != 0 || len(out.Citations) != 0 || out.TokenUsage.Status != "not_called" {
				t.Fatal("source gate failed", out, err)
			}
			if sends.Load() != 0 {
				t.Fatal("source gate generated")
			}
		})
	}
}

func TestS2ManagedConcurrentAndWriteFailure(t *testing.T) {
	var sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sends.Add(1)
		fmt.Fprint(w, syntheticEnvelope(`{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[]}"}}]}`))
	}))
	defer server.Close()
	c, mount, dir := managedTestConfig(t, server.URL, map[string]any{"corpus": fixture(t)})
	var wg sync.WaitGroup
	for range 8 {
		e := managedTestEngine(t, c, mount, dir)
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := e.Query(context.Background(), request())
			if err != nil && ErrorCode(err) != "PREVIOUS_EXECUTION_UNRESOLVED" && ErrorCode(err) != "RECEIPT_CORRUPT" {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	calls, _ := os.ReadFile(filepath.Join(dir, "calls"))
	if sends.Load() != 1 || string(calls) != "call\n" {
		t.Fatal("competing engines initiated duplicate work")
	}
	// Force reservation failure without relying on chmod being enforced for root.
	e := managedTestEngine(t, c, mount, dir)
	moved := c.StoreDir + "-saved"
	if err := os.Rename(c.StoreDir, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.StoreDir, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	r := request()
	r.RequestID = "reservation-failure"
	_, _, err := e.Query(context.Background(), r)
	requireCode(t, err, "STORE_UNAVAILABLE")
	if sends.Load() != 1 {
		t.Fatal("failed reservation generated")
	}
	if err := os.Remove(c.StoreDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(moved, c.StoreDir); err != nil {
		t.Fatal(err)
	}
	// After-response failure keeps the original durable reservation unknown.
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sends.Add(1)
		if err := os.Rename(c.StoreDir, moved); err != nil {
			t.Error(err)
		}
		fmt.Fprint(w, syntheticEnvelope(`{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[]}"}}]}`))
	}))
	defer failing.Close()
	copyManaged := *c.Managed
	copyManaged.DaemonOrigin = failing.URL
	c.Managed = &copyManaged
	t.Setenv("OPENAI_BASE_URL", failing.URL+"/api/runtime/sandboxes/"+strings.Repeat("a", 64)+"/llm/openai/v1")
	e = managedTestEngine(t, c, mount, dir)
	r.RequestID = "finish-failure"
	_, _, err = e.Query(context.Background(), r)
	requireCode(t, err, "STORE_WRITE_FAILED")
	if err := os.Rename(moved, c.StoreDir); err != nil {
		t.Fatal(err)
	}
	e = managedTestEngine(t, c, mount, dir)
	_, _, err = e.Query(context.Background(), r)
	requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
	if sends.Load() != 2 {
		t.Fatal("unknown post-response outcome was retried")
	}
	b, _ := os.ReadFile(filepath.Join(c.StoreDir, r.RequestID+".json"))
	var rec receipt
	if StrictJSON(b, &rec) != nil || rec.State != "reserved" || rec.TokenUsage.Status != "unknown" {
		t.Fatal("failed finish claimed completion or known accounting")
	}
}

func TestS2BuildChangeConflicts(t *testing.T) {
	r := request()
	r.Question = "不存在的测试关键词"
	if path := os.Getenv("CQA_S2_BUILD_PROBE"); path != "" {
		c, err := LoadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		mount := []byte("25 20 0:1 /receipts " + c.StoreDir + " rw - ext4 /dev/test rw\n")
		e, err := newEngine(c, mount)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = e.Query(context.Background(), r)
		requireCode(t, err, "IDEMPOTENCY_CONFLICT")
		return
	}
	c, mount, dir := managedTestConfig(t, "http://127.0.0.1:1", map[string]any{"corpus": fixture(t)})
	e := managedTestEngine(t, c, mount, dir)
	out, _, err := e.Query(context.Background(), r)
	if err != nil || out.Status != "INSUFFICIENT_EVIDENCE" {
		t.Fatal(out, err)
	}
	receiptPath := filepath.Join(c.StoreDir, r.RequestID+".json")
	before, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "config.json")
	b, _ := json.Marshal(c)
	if err := os.WriteFile(cp, b, 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "changed-build.test")
	build := exec.Command("go", "test", "-c", "-trimpath", "-ldflags=-buildid=s2-different-build", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("offline build: %v %s", err, output)
	}
	probe := exec.Command(binary, "-test.run=^TestS2BuildChangeConflicts$")
	probe.Env = append(os.Environ(), "CQA_S2_BUILD_PROBE="+cp)
	if output, err := probe.CombinedOutput(); err != nil {
		t.Fatalf("changed executable did not refuse replay: %v %s", err, output)
	}
	after, _ := os.ReadFile(receiptPath)
	calls, _ := os.ReadFile(filepath.Join(dir, "calls"))
	if !bytes.Equal(before, after) || string(calls) != "call\n" {
		t.Fatal("changed build modified receipt or repeated Search")
	}
}

func TestS2ManagedFailuresRemainUnresolved(t *testing.T) {
	for _, scenario := range []struct{ name, code, usage string }{
		{"http-error", "UPSTREAM_REJECTED", "unknown"},
		{"timeout", "UPSTREAM_OUTCOME_UNKNOWN", "unknown"},
		{"cancel", "UPSTREAM_OUTCOME_UNKNOWN", "unknown"},
		{"forged", "CITATION_NOT_IN_EVIDENCE_SET", "reported"},
		{"wrong-model", "MODEL_IDENTITY_INVALID", "reported"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var sends atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sends.Add(1)
				switch scenario.name {
				case "http-error":
					w.WriteHeader(403)
					fmt.Fprint(w, "PRIVATE_UPSTREAM_ERROR")
					return
				case "timeout":
					time.Sleep(1300 * time.Millisecond)
					return
				case "cancel":
					_, _ = io.Copy(io.Discard, r.Body)
					cancel()
					<-r.Context().Done()
					return
				}
				body := syntheticEnvelope(`{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[{\"text\":\"测试\",\"evidenceIds\":[\"forged\"]}]}"}}]}`)
				if scenario.name == "wrong-model" {
					body = strings.Replace(body, "grok-4.6", "wrong-model", 1)
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			c, mount, dir := managedTestConfig(t, server.URL, map[string]any{"corpus": fixture(t)})
			c.TimeoutSeconds = 1
			e := managedTestEngine(t, c, mount, dir)
			_, _, err := e.Query(ctx, request())
			requireCode(t, err, scenario.code)
			_, _, err = e.Query(context.Background(), request())
			requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
			b, _ := os.ReadFile(filepath.Join(c.StoreDir, request().RequestID+".json"))
			var rec receipt
			if StrictJSON(b, &rec) != nil || rec.State != "blocked" || rec.TokenUsage == nil || rec.TokenUsage.Status != scenario.usage || sends.Load() != 1 || bytes.Contains(b, []byte("PRIVATE_UPSTREAM_ERROR")) {
				t.Fatal("failed request lost usage/privacy or retried")
			}
			calls, _ := os.ReadFile(filepath.Join(dir, "calls"))
			if string(calls) != "call\n" {
				t.Fatal("failed model request repeated Search")
			}
		})
	}
}

func TestS2NativeRefusalsDoNotGenerate(t *testing.T) {
	var sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { sends.Add(1) }))
	defer server.Close()
	for _, exit := range []int{71, 80, 1} {
		c, mount, dir := managedTestConfig(t, server.URL, map[string]any{"corpus": fixture(t)})
		path := filepath.Join(dir, "grpcurl-double")
		if err := os.WriteFile(path, []byte(fmt.Sprintf("#!/bin/sh\nprintf 'attempt\\n' >> \"${0%%/*}/calls\"\nexit %d\n", exit)), 0700); err != nil {
			t.Fatal(err)
		}
		e := managedTestEngine(t, c, mount, dir)
		_, _, err := e.Query(context.Background(), request())
		code := "UPSTREAM_REJECTED"
		if exit == 1 {
			code = "UPSTREAM_OUTCOME_UNKNOWN"
		}
		requireCode(t, err, code)
		_, _, err = e.Query(context.Background(), request())
		requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
		calls, _ := os.ReadFile(filepath.Join(dir, "calls"))
		if string(calls) != "attempt\n" || sends.Load() != 0 {
			t.Fatal("native refusal retried or generated")
		}
	}
}

func TestS2CorruptAndUnresolvedReceiptStates(t *testing.T) {
	c, mount, dir := managedTestConfig(t, "http://127.0.0.1:1", map[string]any{"corpus": fixture(t)})
	e := managedTestEngine(t, c, mount, dir)
	r := request()
	r.Question = "不存在的测试关键词"
	if _, _, err := e.Query(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(c.StoreDir, r.RequestID+".json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"reserved", "blocked", "invalid", "completed-without-result", "completed-without-digest"} {
		var rec receipt
		if StrictJSON(original, &rec) != nil {
			t.Fatal("invalid baseline receipt")
		}
		rec.State = state
		rec.Result, rec.ResultDigest = nil, ""
		code := "RECEIPT_CORRUPT"
		if state == "reserved" || state == "blocked" {
			code = "PREVIOUS_EXECUTION_UNRESOLVED"
		}
		if strings.HasPrefix(state, "completed-") {
			rec.State = "completed"
			if state == "completed-without-digest" {
				var old receipt
				if StrictJSON(original, &old) != nil {
					t.Fatal("invalid baseline receipt")
				}
				rec.Result = old.Result
			}
		}
		b, _ := json.Marshal(rec)
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
		_, _, err := e.Query(context.Background(), r)
		requireCode(t, err, code)
	}
	calls, _ := os.ReadFile(filepath.Join(dir, "calls"))
	if string(calls) != "call\n" {
		t.Fatal("invalid or unresolved receipt repeated Search")
	}
}

func TestS2MountRootReplacementStopsBeforeSend(t *testing.T) {
	c, mount, dir := managedTestConfig(t, "http://127.0.0.1:1", map[string]any{"corpus": fixture(t)})
	e := managedTestEngine(t, c, mount, dir)
	if err := os.Rename(c.StoreDir, c.StoreDir+"-mounted"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(c.StoreDir, 0700); err != nil {
		t.Fatal(err)
	}
	_, _, err := e.Query(context.Background(), request())
	requireCode(t, err, "PERSISTENT_STORE_REQUIRED")
	if _, err := os.Stat(filepath.Join(dir, "calls")); !os.IsNotExist(err) {
		t.Fatal("replaced persistent root allowed Search")
	}
}
