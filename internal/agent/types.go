// Package agent implements a bounded, read-only compliance reference workflow.
// It does not own Accord Cases, approvals, publication, or runtime lifecycle.
package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

const Version = "0.1.0-starter"
const RequestSchema = "cqa.query/v1"
const ResultSchema = "cqa.result/v1"
const SourceSchema = "cqa.corpus/v1"

var safeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,95}$`)
var topics = map[string]bool{"mlps": true, "ciip": true, "security_policy": true, "industry": true, "commercial_crypto": true}

type Request struct {
	SchemaVersion string `json:"schemaVersion"`
	RequestID     string `json:"requestId"`
	Question      string `json:"question"`
	Topic         string `json:"topic"`
	AsOfDate      string `json:"asOfDate"`
	Jurisdiction  string `json:"jurisdiction"`
	Industry      string `json:"industry"`
	CaseID        string `json:"caseId,omitempty"`
	InvocationID  string `json:"invocationId,omitempty"`
	ContextDigest string `json:"contextDigest,omitempty"`
}

type Source struct {
	ID                string   `json:"id"`
	DocumentID        string   `json:"documentId"`
	Title             string   `json:"title"`
	Version           string   `json:"version"`
	Topic             string   `json:"topic"`
	Jurisdiction      string   `json:"jurisdiction"`
	Industry          string   `json:"industry"`
	Publisher         string   `json:"publisher"`
	SourceKind        string   `json:"sourceKind"`
	URI               string   `json:"uri"`
	Locator           string   `json:"locator"`
	PublishedAt       string   `json:"publishedAt"`
	EffectiveFrom     string   `json:"effectiveFrom"`
	EffectiveTo       string   `json:"effectiveTo,omitempty"`
	ValidityCheckedAt string   `json:"validityCheckedAt"`
	Status            string   `json:"status"`
	CuratorReviewed   bool     `json:"curatorReviewed"`
	Synthetic         bool     `json:"synthetic"`
	Keywords          []string `json:"keywords"`
	Text              string   `json:"text"`
	SHA256            string   `json:"sha256"`
}

type Corpus struct {
	SchemaVersion string   `json:"schemaVersion"`
	DatasetID     string   `json:"datasetId"`
	Synthetic     bool     `json:"synthetic"`
	Sources       []Source `json:"sources"`
}

type Claim struct {
	Text        string   `json:"text"`
	EvidenceIDs []string `json:"evidenceIds"`
}

type Citation struct {
	ID                string `json:"id"`
	DocumentID        string `json:"documentId"`
	Title             string `json:"title"`
	Version           string `json:"version"`
	Publisher         string `json:"publisher"`
	SourceKind        string `json:"sourceKind"`
	URI               string `json:"uri"`
	Locator           string `json:"locator"`
	Quote             string `json:"quote"`
	SHA256            string `json:"sha256"`
	PublishedAt       string `json:"publishedAt"`
	EffectiveFrom     string `json:"effectiveFrom"`
	EffectiveTo       string `json:"effectiveTo,omitempty"`
	ValidityCheckedAt string `json:"validityCheckedAt"`
	Synthetic         bool   `json:"synthetic"`
}

type Result struct {
	SchemaVersion       string     `json:"schemaVersion"`
	AgentVersion        string     `json:"agentVersion"`
	RequestID           string     `json:"requestId"`
	CaseID              string     `json:"caseId,omitempty"`
	InvocationID        string     `json:"invocationId,omitempty"`
	ContextDigest       string     `json:"contextDigest,omitempty"`
	InputDigest         string     `json:"inputDigest"`
	ConfigDigest        string     `json:"configDigest"`
	CorpusDigest        string     `json:"corpusDigest"`
	Status              string     `json:"status"`
	DataMode            string     `json:"dataMode"`
	GenerationMode      string     `json:"generationMode"`
	AsOfDate            string     `json:"asOfDate"`
	DatasetID           string     `json:"datasetId"`
	Claims              []Claim    `json:"claims"`
	Citations           []Citation `json:"citations"`
	ReasonCodes         []string   `json:"reasonCodes"`
	Warnings            []string   `json:"warnings"`
	HumanReviewRequired bool       `json:"humanReviewRequired"`
	EntailmentVerified  bool       `json:"entailmentVerified"`
	GeneratedAt         string     `json:"generatedAt"`
}

type Error struct {
	Code       string
	HTTPStatus int
}

func (e *Error) Error() string              { return e.Code }
func problem(code string, status int) error { return &Error{code, status} }
func ErrorCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "INTERNAL_ERROR"
}
func HTTPStatus(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.HTTPStatus
	}
	return 500
}
func SHA(data []byte) string { d := sha256.Sum256(data); return hex.EncodeToString(d[:]) }
func hashJSON(v any) string  { b, _ := json.Marshal(v); return SHA(b) }
func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && len(s) == 10
}

// StrictJSON rejects duplicate keys, unknown fields, multiple values and deep shapes.
func StrictJSON(data []byte, v any) error {
	d := json.NewDecoder(strings.NewReader(string(data)))
	var visit func(int) error
	nodes := 0
	visit = func(depth int) error {
		nodes++
		if depth > 20 || nodes > 100000 {
			return errors.New("JSON bounds")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				key, ok := k.(string)
				if !ok || seen[key] {
					return errors.New("duplicate key")
				}
				seen[key] = true
				if e = visit(depth + 1); e != nil {
					return e
				}
			}
			_, err = d.Token()
			return err
		case '[':
			for d.More() {
				if err = visit(depth + 1); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		default:
			return errors.New("bad delimiter")
		}
	}
	if err := visit(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	d = json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	return nil
}

func (r Request) Validate() error {
	if r.SchemaVersion != RequestSchema || !safeID.MatchString(r.RequestID) || len(strings.TrimSpace(r.Question)) < 2 || len(r.Question) > 8000 || !validDate(r.AsOfDate) || r.Jurisdiction != "CN" || len(r.Industry) < 1 || len(r.Industry) > 80 {
		return problem("INVALID_REQUEST", 400)
	}
	if !topics[r.Topic] {
		return problem("TOPIC_UNSUPPORTED", 422)
	}
	if r.CaseID != "" || r.InvocationID != "" || r.ContextDigest != "" {
		b, e := hex.DecodeString(r.ContextDigest)
		if !safeID.MatchString(r.CaseID) || !safeID.MatchString(r.InvocationID) || e != nil || len(b) != 32 {
			return problem("INVALID_ACCORD_CORRELATION", 400)
		}
	}
	return nil
}
