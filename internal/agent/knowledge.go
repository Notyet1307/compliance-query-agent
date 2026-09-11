package agent

import (
	"context"
	"net/url"
	"os"
	"sort"
	"strings"
)

type Knowledge interface {
	Search(context.Context, Request) (Corpus, error)
}
type localKnowledge struct{ corpus Corpus }

func (l localKnowledge) Search(ctx context.Context, r Request) (Corpus, error) {
	if ctx.Err() != nil {
		return Corpus{}, problem("DEADLINE_EXCEEDED", 504)
	}
	return l.corpus, nil
}

func LoadCorpus(path string, allowSynthetic bool) (Corpus, error) {
	var c Corpus
	b, e := os.ReadFile(path)
	if e != nil || len(b) > 4*1024*1024 {
		return c, problem("CORPUS_UNREADABLE", 500)
	}
	if StrictJSON(b, &c) != nil {
		return c, problem("CORPUS_INVALID", 502)
	}
	return c, validateCorpus(c, allowSynthetic)
}
func validateCorpus(c Corpus, allow bool) error {
	if c.SchemaVersion != SourceSchema || !safeID.MatchString(c.DatasetID) || len(c.Sources) > 2000 {
		return problem("CORPUS_INVALID", 502)
	}
	if c.Synthetic && !allow {
		return problem("SYNTHETIC_DATA_DENIED", 403)
	}
	ids := map[string]bool{}
	for _, s := range c.Sources {
		if !safeID.MatchString(s.ID) || ids[s.ID] || !safeID.MatchString(s.DocumentID) || s.Title == "" || len(s.Title) > 500 || s.Version == "" || s.Locator == "" || s.Publisher == "" || !topics[s.Topic] || s.Jurisdiction != "CN" || s.Industry == "" || len(s.Text) == 0 || len(s.Text) > 12000 || len(s.Keywords) == 0 || len(s.Keywords) > 32 || s.Synthetic != c.Synthetic {
			return problem("SOURCE_INVALID", 502)
		}
		ids[s.ID] = true
		if s.SHA256 != SHA([]byte(s.Text)) {
			return problem("SOURCE_DIGEST_MISMATCH", 502)
		}
		if !validDate(s.PublishedAt) || !validDate(s.EffectiveFrom) || !validDate(s.ValidityCheckedAt) || (s.EffectiveTo != "" && (!validDate(s.EffectiveTo) || s.EffectiveTo <= s.EffectiveFrom)) {
			return problem("SOURCE_DATE_INVALID", 502)
		}
		if s.Status != "in_force" && s.Status != "repealed" && s.Status != "draft" && s.Status != "unknown" {
			return problem("SOURCE_STATUS_INVALID", 502)
		}
		if s.Status == "repealed" && s.EffectiveTo == "" {
			return problem("SOURCE_INTERVAL_REQUIRED", 502)
		}
		if s.ValidityCheckedAt < s.PublishedAt {
			return problem("SOURCE_DATE_INVALID", 502)
		}
		for _, k := range s.Keywords {
			if len(strings.TrimSpace(k)) < 2 || len(k) > 120 {
				return problem("SOURCE_KEYWORDS_INVALID", 502)
			}
		}
		if s.Synthetic {
			if s.SourceKind != "synthetic" || !strings.HasPrefix(s.URI, "fixture://") {
				return problem("SYNTHETIC_LABEL_INVALID", 502)
			}
		} else {
			if s.SourceKind != "official" && s.SourceKind != "licensed_standard" && s.SourceKind != "internal" {
				return problem("SOURCE_KIND_INVALID", 502)
			}
			u, e := url.Parse(s.URI)
			if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
				return problem("SOURCE_URI_INVALID", 502)
			}
		}
	}
	return nil
}

type selection struct {
	sources []Source
	reasons []string
	blocked bool
}

func selectSources(c Corpus, r Request) selection {
	type ranked struct {
		s     Source
		score int
	}
	var list []ranked
	reasons := map[string]bool{}
	versions := map[string]string{}
	conflict := false
	for _, s := range c.Sources {
		if s.Topic != r.Topic || s.Jurisdiction != r.Jurisdiction || (s.Industry != "all" && s.Industry != r.Industry) {
			continue
		}
		score := 0
		for _, k := range s.Keywords {
			if strings.Contains(strings.ToLower(r.Question), strings.ToLower(k)) {
				score++
			}
		}
		if score == 0 {
			continue
		}
		if s.PublishedAt > r.AsOfDate || s.EffectiveFrom > r.AsOfDate || (s.EffectiveTo != "" && r.AsOfDate >= s.EffectiveTo) {
			reasons["OUTSIDE_EFFECTIVE_INTERVAL"] = true
			continue
		}
		if s.Status == "draft" || s.Status == "unknown" || !s.CuratorReviewed {
			reasons["SOURCE_NOT_REVIEWED_OR_NOT_EFFECTIVE"] = true
			continue
		}
		if s.ValidityCheckedAt < r.AsOfDate {
			reasons["CURRENCY_UNVERIFIED"] = true
			continue
		}
		key := s.DocumentID + "\x00" + s.Locator
		value := s.Version + "\x00" + s.SHA256
		if old, ok := versions[key]; ok && old != value {
			conflict = true
		}
		versions[key] = value
		list = append(list, ranked{s, score})
	}
	out := selection{sources: []Source{}, reasons: []string{}}
	if conflict {
		out.blocked = true
		reasons["OVERLAPPING_SOURCE_VERSIONS"] = true
	}
	// Fail closed when potentially relevant, stale/unreviewed sources could change the answer.
	if reasons["CURRENCY_UNVERIFIED"] || reasons["SOURCE_NOT_REVIEWED_OR_NOT_EFFECTIVE"] {
		out.blocked = true
	}
	for k := range reasons {
		out.reasons = append(out.reasons, k)
	}
	sort.Strings(out.reasons)
	sort.Slice(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score > list[j].score
		}
		return list[i].s.ID < list[j].s.ID
	})
	for i, x := range list {
		if i == 5 {
			break
		}
		out.sources = append(out.sources, x.s)
	}
	return out
}
func citation(s Source) Citation {
	return Citation{ID: s.ID, DocumentID: s.DocumentID, Title: s.Title, Version: s.Version, Publisher: s.Publisher, SourceKind: s.SourceKind, URI: s.URI, Locator: s.Locator, Quote: s.Text, SHA256: s.SHA256, PublishedAt: s.PublishedAt, EffectiveFrom: s.EffectiveFrom, EffectiveTo: s.EffectiveTo, ValidityCheckedAt: s.ValidityCheckedAt, Synthetic: s.Synthetic}
}
