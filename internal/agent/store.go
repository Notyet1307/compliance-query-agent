package agent

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

type receipt struct {
	SchemaVersion string      `json:"schemaVersion"`
	InputDigest   string      `json:"inputDigest"`
	ConfigDigest  string      `json:"configDigest"`
	State         string      `json:"state"`
	Result        *Result     `json:"result,omitempty"`
	ErrorCode     string      `json:"errorCode,omitempty"`
	TokenUsage    *TokenUsage `json:"tokenUsage,omitempty"`
}
type Store struct{ dir string }

func NewStore(dir string) (*Store, error) {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, problem("STORE_UNAVAILABLE", 500)
	}
	st, e := os.Lstat(dir)
	if e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, problem("STORE_PERMISSIONS_UNSAFE", 500)
	}
	return &Store{dir}, nil
}

// O_EXCL reserves a stable request before network I/O. An interrupted reservation
// never auto-retries. This is a local execution receipt, not an Accord task ledger.
func (s *Store) reserve(id, input, config string) (*Result, error) {
	p := filepath.Join(s.dir, id+".json")
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(e) {
		st, e := os.Lstat(p)
		if e != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 {
			return nil, problem("RECEIPT_UNSAFE", 500)
		}
		b, e := os.ReadFile(p)
		var old receipt
		if e != nil || len(b) > 512*1024 || StrictJSON(b, &old) != nil || old.SchemaVersion != "cqa.receipt/v1" {
			return nil, problem("RECEIPT_CORRUPT", 500)
		}
		if old.InputDigest != input || old.ConfigDigest != config {
			return nil, problem("IDEMPOTENCY_CONFLICT", 409)
		}
		if old.State == "completed" && old.Result != nil {
			r := old.Result
			if r.SchemaVersion != ResultSchema || r.RequestID != id || r.InputDigest != input || r.ConfigDigest != config {
				return nil, problem("RECEIPT_CORRUPT", 500)
			}
			return r, nil
		}
		return nil, problem("PREVIOUS_EXECUTION_UNRESOLVED", 409)
	}
	if e != nil {
		return nil, problem("STORE_UNAVAILABLE", 500)
	}
	record := receipt{SchemaVersion: "cqa.receipt/v1", InputDigest: input, ConfigDigest: config, State: "reserved", TokenUsage: &TokenUsage{Status: "unknown"}}
	e = json.NewEncoder(f).Encode(record)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return nil, problem("STORE_WRITE_FAILED", 500)
	}
	return nil, s.syncDir()
}
func (s *Store) syncDir() error {
	f, e := os.Open(s.dir)
	if e != nil {
		return problem("STORE_WRITE_FAILED", 500)
	}
	defer f.Close()
	if f.Sync() != nil {
		return problem("STORE_WRITE_FAILED", 500)
	}
	return nil
}
func (s *Store) finish(id, input, config string, r *Result, usage *TokenUsage, runErr error) error {
	rec := receipt{SchemaVersion: "cqa.receipt/v1", InputDigest: input, ConfigDigest: config, State: "completed", Result: r}
	if runErr != nil {
		rec.State = "blocked"
		rec.ErrorCode = ErrorCode(runErr)
		rec.Result = nil
		rec.TokenUsage = usage
	}
	b, e := json.Marshal(rec)
	if e != nil {
		return problem("STORE_WRITE_FAILED", 500)
	}
	f, e := os.CreateTemp(s.dir, ".receipt-*")
	if e != nil {
		return problem("STORE_WRITE_FAILED", 500)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	n, e := f.Write(b)
	if e == nil && n != len(b) {
		e = io.ErrShortWrite
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return problem("STORE_WRITE_FAILED", 500)
	}
	if os.Rename(tmp, filepath.Join(s.dir, id+".json")) != nil {
		return problem("STORE_WRITE_FAILED", 500)
	}
	return s.syncDir()
}
