package agent

import (
	"context"
	"os"
	"strings"
	"time"
)

type Engine struct {
	cfg          Config
	knowledge    Knowledge
	generator    Generator
	store        *Store
	configDigest string
	now          func() time.Time
}

func NewEngine(c Config) (*Engine, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	var mountInfo []byte
	if c.Managed != nil {
		var err error
		mountInfo, err = os.ReadFile("/proc/self/mountinfo")
		if err != nil {
			return nil, problem("PERSISTENT_STORE_REQUIRED", 500)
		}
	}
	return newEngine(c, mountInfo)
}

func newEngine(c Config, mountInfo []byte) (*Engine, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	var s *Store
	var err error
	build := ""
	if c.Managed != nil {
		s, err = managedStore(c.StoreDir, mountInfo)
		if err == nil {
			build, err = executableDigest()
		}
	} else {
		s, err = NewStore(c.StoreDir)
	}
	if err != nil {
		return nil, err
	}
	var k Knowledge
	var g Generator
	localDigest := "external-at-query-time"
	if c.Knowledge.Mode == "local" {
		corpus, e := LoadCorpus(c.Knowledge.CorpusPath, c.AllowSynthetic)
		if e != nil {
			return nil, e
		}
		k = localKnowledge{corpus}
		localDigest = hashJSON(corpus)
	} else if c.Knowledge.Mode == "octobus_native_x1" || c.Knowledge.Mode == "octobus_native_s2" {
		native, err := newNativeKnowledge(c)
		if err != nil {
			return nil, err
		}
		k = native
	} else {
		token := os.Getenv(c.Knowledge.TokenEnv)
		if len(token) < 16 || strings.ContainsAny(token, "\r\n") {
			return nil, problem("OCTOBUS_CREDENTIAL_MISSING", 500)
		}
		k = octobusKnowledge{c.Knowledge.Endpoint, token, newHTTPClient(c.TimeoutSeconds)}
	}
	if c.Generation.Mode == "extractive" {
		g = extractiveGenerator{}
	} else {
		generation := c.Generation
		if c.Managed != nil {
			generation, err = managedGeneration(c)
			if err != nil {
				return nil, err
			}
		}
		token := os.Getenv(c.Generation.TokenEnv)
		if len(token) < 16 || strings.ContainsAny(token, "\r\n\t ") {
			return nil, problem("MODEL_CREDENTIAL_MISSING", 500)
		}
		g = llmGenerator{generation, token, newHTTPClient(c.TimeoutSeconds)}
	}
	// A rotated credential value is not hashed into public output. Stable ref is part of config.
	d := hashJSON(struct {
		Config           Config
		CorpusDigest     string
		AgentVersion     string
		ExecutableSHA256 string `json:",omitempty"`
	}{c, localDigest, Version, build})
	return &Engine{c, k, g, s, d, time.Now}, nil
}
func (e *Engine) Query(ctx context.Context, r Request) (*Result, bool, error) {
	if err := r.Validate(); err != nil {
		return nil, false, err
	}
	input := hashJSON(r)
	prior, err := e.store.reserve(r.RequestID, input, e.configDigest)
	if err != nil {
		return nil, false, err
	}
	if prior != nil {
		return prior, true, nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	usage := TokenUsage{Status: "not_called"}
	result, runErr := e.execute(ctx, r, input, &usage)
	if err = e.store.finish(r.RequestID, input, e.configDigest, result, &usage, runErr); err != nil {
		return nil, false, err
	}
	return result, false, runErr
}
func (e *Engine) execute(ctx context.Context, r Request, input string, usage *TokenUsage) (*Result, error) {
	corpus, err := e.knowledge.Search(ctx, r)
	if err != nil {
		return nil, err
	}
	if err = validateCorpus(corpus, e.cfg.AllowSynthetic); err != nil {
		return nil, err
	}
	mode := "reviewed_snapshot"
	if corpus.Synthetic {
		mode = "synthetic_demo"
	}
	out := &Result{SchemaVersion: ResultSchema, AgentVersion: Version, RequestID: r.RequestID, CaseID: r.CaseID, InvocationID: r.InvocationID, ContextDigest: r.ContextDigest, InputDigest: input, ConfigDigest: e.configDigest, CorpusDigest: hashJSON(corpus), Status: "INSUFFICIENT_EVIDENCE", DataMode: mode, GenerationMode: e.cfg.Generation.Mode, AsOfDate: r.AsOfDate, DatasetID: corpus.DatasetID, Claims: []Claim{}, Citations: []Citation{}, ReasonCodes: []string{}, Warnings: []string{"仅为资料查询/参考草稿，不是法律意见、测评结论或认证。", "validityCheckedAt 是资料管理员的核验记录，不代表本程序实时查询了官方网站。", "当前检索为关键词匹配，不保证知识覆盖完整；未实现跨文件语义冲突识别。"}, HumanReviewRequired: true, EntailmentVerified: false, GeneratedAt: e.now().UTC().Format(time.RFC3339)}
	out.TokenUsage = usage
	if corpus.Synthetic {
		out.Warnings = append(out.Warnings, "全部来源均为虚构测试材料，不是现行法规或标准，不得用于实际合规判断。")
	}
	selected := selectSources(corpus, r)
	out.ReasonCodes = selected.reasons
	if selected.blocked {
		out.Status = "NEEDS_REVIEW"
		return out, nil
	}
	if len(selected.sources) == 0 {
		out.ReasonCodes = append(out.ReasonCodes, "NO_ELIGIBLE_EVIDENCE")
		return out, nil
	}
	claims, reported, err := e.generator.Generate(ctx, r, selected.sources)
	*usage = reported
	if err != nil {
		return nil, err
	}
	if err = validateClaims(claims, selected.sources); err != nil {
		return nil, err
	}
	if len(claims) == 0 {
		out.ReasonCodes = append(out.ReasonCodes, "MODEL_REPORTED_INSUFFICIENT_EVIDENCE")
		return out, nil
	}
	out.Status = "REFERENCE_ONLY"
	if e.cfg.Generation.Mode == "llm" {
		out.Status = "DRAFT_READY"
	}
	out.Claims = claims
	used := map[string]bool{}
	for _, c := range claims {
		for _, id := range c.EvidenceIDs {
			used[id] = true
		}
	}
	for _, s := range selected.sources {
		if used[s.ID] {
			out.Citations = append(out.Citations, citation(s))
		}
	}
	return out, nil
}
