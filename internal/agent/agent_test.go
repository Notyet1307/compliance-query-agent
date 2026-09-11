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
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func fixture(t *testing.T) Corpus {
	t.Helper()
	c, e := LoadCorpus("../../testdata/demo-corpus.json", true)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func config(t *testing.T) Config {
	t.Helper()
	p, e := filepath.Abs("../../testdata/demo-corpus.json")
	if e != nil {
		t.Fatal(e)
	}
	return Config{SchemaVersion: "cqa.config/v1", StoreDir: filepath.Join(t.TempDir(), "receipts"), AllowSynthetic: true, TimeoutSeconds: 3, Knowledge: KnowledgeConfig{Mode: "local", CorpusPath: p}, Generation: GenerationConfig{Mode: "extractive"}, Server: ServerConfig{Listen: "127.0.0.1:8088", TokenEnv: "TEST_API"}}
}
func request() Request {
	return Request{SchemaVersion: RequestSchema, RequestID: "test-1", Question: "资产清单需要记录什么", Topic: "mlps", AsOfDate: "2026-09-10", Jurisdiction: "CN", Industry: "telecom"}
}
func engine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine(config(t))
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || ErrorCode(err) != code {
		t.Fatalf("got %v want %s", err, code)
	}
}

func TestFiveDomainsAndExceptions(t *testing.T) {
	tests := []struct{ name, topic, q, status, reason string }{
		{"mlps", "mlps", "资产清单", "REFERENCE_ONLY", ""},
		{"ciip", "ciip", "应急联系人", "REFERENCE_ONLY", ""},
		{"policy", "security_policy", "制度版本", "REFERENCE_ONLY", ""},
		{"industry", "industry", "运营商术语", "REFERENCE_ONLY", ""},
		{"crypto", "commercial_crypto", "密码应用台账", "REFERENCE_ONLY", ""},
		{"no-evidence", "mlps", "三级系统的审计日志应保留多久", "INSUFFICIENT_EVIDENCE", "NO_ELIGIBLE_EVIDENCE"},
		{"stale", "security_policy", "过期核验测试", "NEEDS_REVIEW", "CURRENCY_UNVERIFIED"},
		{"conflict", "mlps", "重叠版本测试", "NEEDS_REVIEW", "OVERLAPPING_SOURCE_VERSIONS"},
		{"future", "security_policy", "未来条目测试", "INSUFFICIENT_EVIDENCE", "OUTSIDE_EFFECTIVE_INTERVAL"},
	}
	for _, x := range tests {
		t.Run(x.name, func(t *testing.T) {
			e := engine(t)
			r := request()
			r.Topic = x.topic
			r.Question = x.q
			out, _, err := e.Query(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			if out.Status != x.status {
				t.Fatalf("%+v", out)
			}
			if out.DataMode != "synthetic_demo" || !out.HumanReviewRequired || out.EntailmentVerified {
				t.Fatal("misleading trust")
			}
			if x.reason != "" && !strings.Contains(strings.Join(out.ReasonCodes, ","), x.reason) {
				t.Fatal(out.ReasonCodes)
			}
			for _, c := range out.Citations {
				if c.SHA256 != SHA([]byte(c.Quote)) || !c.Synthetic {
					t.Fatal("bad citation")
				}
			}
		})
	}
}
func TestNoEvidenceDoesNotCallLLM(t *testing.T) {
	e := engine(t)
	g := &countGenerator{}
	e.generator = g
	r := request()
	r.Question = "审计日志保存时长"
	_, _, err := e.Query(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if g.calls.Load() != 0 {
		t.Fatal("called model without evidence")
	}
}

type countGenerator struct{ calls atomic.Int32 }

func (g *countGenerator) Generate(ctx context.Context, r Request, s []Source) ([]Claim, error) {
	g.calls.Add(1)
	time.Sleep(20 * time.Millisecond)
	return extractiveGenerator{}.Generate(ctx, r, s)
}
func TestIdempotencyAndConflict(t *testing.T) {
	e := engine(t)
	g := &countGenerator{}
	e.generator = g
	r := request()
	out, rep, err := e.Query(context.Background(), r)
	if err != nil || rep {
		t.Fatal(err)
	}
	again, rep, err := e.Query(context.Background(), r)
	if err != nil || !rep || hashJSON(out) != hashJSON(again) || g.calls.Load() != 1 {
		t.Fatal("not stable replay", err)
	}
	r.Question = "变更资产清单"
	_, _, err = e.Query(context.Background(), r)
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
}
func TestConcurrentReservation(t *testing.T) {
	e := engine(t)
	g := &countGenerator{}
	e.generator = g
	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := e.Query(context.Background(), request())
			if err == nil {
				successes.Add(1)
			} else if ErrorCode(err) != "PREVIOUS_EXECUTION_UNRESOLVED" && ErrorCode(err) != "RECEIPT_CORRUPT" {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if g.calls.Load() != 1 || successes.Load() < 1 {
		t.Fatal("duplicate execution", g.calls.Load())
	}
}
func TestRestartReplaysReceipt(t *testing.T) {
	c := config(t)
	e, err := NewEngine(c)
	if err != nil {
		t.Fatal(err)
	}
	a, _, err := e.Query(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	next, err := NewEngine(c)
	if err != nil {
		t.Fatal(err)
	}
	b, replay, err := next.Query(context.Background(), request())
	if err != nil || !replay || hashJSON(a) != hashJSON(b) {
		t.Fatal("restart replay failed", err)
	}
}
func TestReservedNeverRetries(t *testing.T) {
	e := engine(t)
	r := request()
	_, err := e.store.reserve(r.RequestID, hashJSON(r), e.configDigest)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = e.Query(context.Background(), r)
	requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
}
func TestConfigDrift(t *testing.T) {
	c := config(t)
	e, _ := NewEngine(c)
	_, _, err := e.Query(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	c.TimeoutSeconds++
	e, _ = NewEngine(c)
	_, _, err = e.Query(context.Background(), request())
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
}
func TestDigestTampering(t *testing.T) {
	c := fixture(t)
	c.Sources[0].Text += "tamper"
	requireCode(t, validateCorpus(c, true), "SOURCE_DIGEST_MISMATCH")
}
func TestSyntheticDenied(t *testing.T) {
	c := config(t)
	c.AllowSynthetic = false
	_, err := NewEngine(c)
	requireCode(t, err, "SYNTHETIC_DATA_DENIED")
}
func TestUnknownTopicAndBadDate(t *testing.T) {
	r := request()
	r.Topic = "investment"
	requireCode(t, r.Validate(), "TOPIC_UNSUPPORTED")
	r = request()
	r.AsOfDate = "2026-02-30"
	requireCode(t, r.Validate(), "INVALID_REQUEST")
}
func TestStrictJSON(t *testing.T) {
	for _, raw := range []string{`{"requestId":"a","requestId":"b"}`, `{"unknown":1}`, `{} {}`, `{"requestId":42}`} {
		var r Request
		if StrictJSON([]byte(raw), &r) == nil {
			t.Fatal("accepted invalid JSON", raw)
		}
	}
}
func TestCorrelationBinding(t *testing.T) {
	r := request()
	r.CaseID = "case-a"
	requireCode(t, r.Validate(), "INVALID_ACCORD_CORRELATION")
	r.InvocationID = "inv-a"
	r.ContextDigest = strings.Repeat("a", 64)
	if r.Validate() != nil {
		t.Fatal("valid correlation rejected")
	}
	e := engine(t)
	out, _, err := e.Query(context.Background(), r)
	if err != nil || out.CaseID != r.CaseID || out.InvocationID != r.InvocationID || out.ContextDigest != r.ContextDigest {
		t.Fatal("binding lost", err)
	}
}
func TestHistoricalInterval(t *testing.T) {
	e := engine(t)
	r := request()
	r.Topic = "commercial_crypto"
	r.Question = "历史条目测试"
	r.AsOfDate = "2026-05-01"
	out, _, err := e.Query(context.Background(), r)
	if err != nil || out.Status != "REFERENCE_ONLY" {
		t.Fatal(out, err)
	}
	r.RequestID = "at-exclusive-end"
	r.AsOfDate = "2026-06-01"
	out, _, err = e.Query(context.Background(), r)
	if err != nil || out.Status != "INSUFFICIENT_EVIDENCE" {
		t.Fatal(out, err)
	}
}
func TestIndustryIsolation(t *testing.T) {
	e := engine(t)
	r := request()
	r.Topic = "industry"
	r.Industry = "finance"
	r.Question = "运营商术语"
	out, _, err := e.Query(context.Background(), r)
	if err != nil || out.Status != "INSUFFICIENT_EVIDENCE" {
		t.Fatal("industry scope crossed", err)
	}
}
func TestUnreviewedSourceBlocks(t *testing.T) {
	c := fixture(t)
	c.Sources[0].CuratorReviewed = false
	s := selectSources(c, request())
	if !s.blocked {
		t.Fatal("unreviewed source accepted")
	}
}
func TestModelCitationValidation(t *testing.T) {
	ss := fixture(t).Sources[:1]
	requireCode(t, validateClaims([]Claim{{"不存在的依据", []string{"made-up"}}}, ss), "CITATION_NOT_IN_EVIDENCE_SET")
	requireCode(t, validateClaims([]Claim{{"没有引用", nil}}, ss), "CLAIM_INVALID")
}
func TestNetworkApprovalAndTLS(t *testing.T) {
	c := config(t)
	c.Generation = GenerationConfig{Mode: "llm", Endpoint: "https://example.invalid/v1/chat/completions", Model: "test-model", TokenEnv: "TEST_TOKEN", MaxTokens: 2048}
	requireCode(t, c.Validate(), "NETWORK_NOT_APPROVED")
	c.NetworkApproved = true
	c.Generation.Endpoint = "http://192.0.2.1/v1/chat/completions"
	requireCode(t, c.Validate(), "TLS_REQUIRED")
}
func TestServerAuthAndLoopback(t *testing.T) {
	e := engine(t)
	token := strings.Repeat("x", 32)
	handler := e.Handler(token)
	b, _ := json.Marshal(request())
	req := httptest.NewRequest("POST", "/v1/query", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != 401 {
		t.Fatal(res.Code)
	}
	req = httptest.NewRequest("POST", "/v1/query", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatal(res.Code, res.Body.String())
	}
	e.cfg.Server.Listen = "0.0.0.0:8088"
	_, err := e.Server(token)
	requireCode(t, err, "LOOPBACK_SERVER_ONLY")
}
func TestAPIRejectsOversizedBody(t *testing.T) {
	e := engine(t)
	token := strings.Repeat("x", 32)
	req := httptest.NewRequest("POST", "/v1/query", strings.NewReader(strings.Repeat("x", 70000)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	e.Handler(token).ServeHTTP(res, req)
	if res.Code != 413 {
		t.Fatal(res.Code)
	}
}
func TestModelHTTPContract(t *testing.T) {
	token := "SYNTHETIC_MODEL_TOKEN_123456"
	t.Setenv("TEST_LLM_TOKEN", token)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("wrong wire")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("decode")
		}
		if body["model"] != "contract-test" || body["stream"] != false {
			t.Error("model mismatch")
		}
		raw, _ := json.Marshal(body)
		if strings.Contains(string(raw), "case-a") || strings.Contains(string(raw), token) || strings.Contains(string(raw), "inv-a") {
			t.Error("unnecessary identifiers or credential in payload")
		}
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[{\"text\":\"这是根据样本形成的草稿。\",\"evidenceIds\":[\"demo-mlps-assets\"]}]}"}}]}`)
	}))
	defer server.Close()
	c := config(t)
	c.NetworkApproved = true
	c.Generation = GenerationConfig{Mode: "llm", Endpoint: server.URL + "/chat/completions", Model: "contract-test", TokenEnv: "TEST_LLM_TOKEN", MaxTokens: 2048}
	e, err := NewEngine(c)
	if err != nil {
		t.Fatal(err)
	}
	r := request()
	r.CaseID = "case-a"
	r.InvocationID = "inv-a"
	r.ContextDigest = strings.Repeat("a", 64)
	out, _, err := e.Query(context.Background(), r)
	if err != nil || out.Status != "DRAFT_READY" || out.EntailmentVerified {
		t.Fatal(out, err)
	}
	_, _, err = e.Query(context.Background(), r)
	if err != nil || calls.Load() != 1 {
		t.Fatal("repeated model", err)
	}
}
func TestModelRejectsForgedCitationAndDoesNotRetry(t *testing.T) {
	token := "SYNTHETIC_MODEL_TOKEN_123456"
	t.Setenv("TEST_LLM_TOKEN", token)
	var n atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"{\"claims\":[{\"text\":\"bad\",\"evidenceIds\":[\"fake\"]}]}"}}]}`)
	}))
	defer server.Close()
	c := config(t)
	c.NetworkApproved = true
	c.Generation = GenerationConfig{Mode: "llm", Endpoint: server.URL, Model: "contract", TokenEnv: "TEST_LLM_TOKEN", MaxTokens: 2048}
	e, err := NewEngine(c)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = e.Query(context.Background(), request())
	requireCode(t, err, "CITATION_NOT_IN_EVIDENCE_SET")
	_, _, err = e.Query(context.Background(), request())
	requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
	if n.Load() != 1 {
		t.Fatal("auto retry")
	}
}
func TestCredentialReflectionBlocked(t *testing.T) {
	token := "SYNTHETIC_MODEL_TOKEN_123456"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, token) }))
	defer server.Close()
	_, err := post(context.Background(), newHTTPClient(2), server.URL, token, map[string]string{}, false)
	requireCode(t, err, "CREDENTIAL_REFLECTION_BLOCKED")
	if strings.Contains(err.Error(), token) {
		t.Fatal("credential leaked")
	}
}
func TestNoRedirect(t *testing.T) {
	var n atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n.Add(1) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer redirect.Close()
	_, err := post(context.Background(), newHTTPClient(2), redirect.URL, "SYNTHETIC_TOKEN_123456", map[string]string{}, false)
	requireCode(t, err, "UPSTREAM_REJECTED")
	if n.Load() != 0 {
		t.Fatal("followed redirect")
	}
}
func TestOctobusConnectContract(t *testing.T) {
	corpus := fixture(t)
	token := "SYNTHETIC_OCTOBUS_TOKEN_123456"
	t.Setenv("TEST_OCTO_TOKEN", token)
	path := "/capsets/compliance-readonly/connect/compliance-kb/compliance.v1.KnowledgeService/Search"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path || r.Header.Get("Connect-Protocol-Version") != "1" || r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("bad Connect shape")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["query"] != request().Question || body["schemaVersion"] != "cqa.search/v1" || body["caseId"] != nil {
			t.Error("bad body")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"corpus": corpus, "truncated": false})
	}))
	defer server.Close()
	c := config(t)
	c.NetworkApproved = true
	c.Knowledge = KnowledgeConfig{Mode: "octobus_connect", Endpoint: server.URL + path, TokenEnv: "TEST_OCTO_TOKEN"}
	e, err := NewEngine(c)
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := e.Query(context.Background(), request())
	if err != nil || out.Status != "REFERENCE_ONLY" {
		t.Fatal(out, err)
	}
}
func TestOctobusTruncationBlocks(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"truncated":true,"corpus":{"schemaVersion":"cqa.corpus/v1","datasetId":"test","synthetic":true,"sources":[]}}`)
	}))
	defer s.Close()
	o := octobusKnowledge{s.URL, "SYNTHETIC_OCTOBUS_TOKEN_123456", newHTTPClient(2)}
	_, err := o.Search(context.Background(), request())
	requireCode(t, err, "RETRIEVAL_TRUNCATED")
}
func TestDeadlineAndReceipt(t *testing.T) {
	e := engine(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := e.Query(ctx, request())
	requireCode(t, err, "DEADLINE_EXCEEDED")
	_, _, err = e.Query(context.Background(), request())
	requireCode(t, err, "PREVIOUS_EXECUTION_UNRESOLVED")
}
func TestStorePermissionsAndPath(t *testing.T) {
	e := engine(t)
	r := request()
	r.RequestID = "../../other"
	_, _, err := e.Query(context.Background(), r)
	requireCode(t, err, "INVALID_REQUEST")
	r = request()
	_, _, err = e.Query(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(e.cfg.StoreDir, r.RequestID+".json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("unsafe receipt", err)
	}
}
func TestLoopbackHTTPAllowedOnlyExplicitly(t *testing.T) {
	if validateEndpoint("http://127.0.0.1:9999/path") != nil {
		t.Fatal("loopback invalid")
	}
	for _, v := range []string{"https://user:secret@example.test/x", "https://example.test/x?token=y", "file:///etc/passwd", "http://localhost:9999/x", "https://example.test/x#secret"} {
		if validateEndpoint(v) == nil {
			t.Fatal("unsafe endpoint", v)
		}
	}
}
