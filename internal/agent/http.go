package agent

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"
)

func (e *Engine) Handler(token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "up", "agentVersion": Version, "meaning": "process_only_not_external_integration"})
	})
	mux.HandleFunc("POST /v1/query", func(w http.ResponseWriter, r *http.Request) {
		if len(token) < 32 || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			writeJSON(w, 401, map[string]string{"error": "UNAUTHORIZED"})
			return
		}
		mediaType, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if mediaErr != nil || mediaType != "application/json" {
			writeJSON(w, 415, map[string]string{"error": "JSON_REQUIRED"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 65536)
		b, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, 413, map[string]string{"error": "REQUEST_TOO_LARGE"})
			return
		}
		var req Request
		if StrictJSON(b, &req) != nil {
			writeJSON(w, 400, map[string]string{"error": "INVALID_JSON"})
			return
		}
		result, replayed, err := e.Query(r.Context(), req)
		if err != nil {
			writeJSON(w, HTTPStatus(err), map[string]string{"error": ErrorCode(err)})
			return
		}
		if replayed {
			w.Header().Set("X-CQA-Replayed", "true")
		}
		writeJSON(w, 200, result)
	})
	return mux
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (e *Engine) Server(token string) (*http.Server, error) {
	host, _, err := net.SplitHostPort(e.cfg.Server.Listen)
	ip := net.ParseIP(host)
	if err != nil || ip == nil || !ip.IsLoopback() {
		return nil, problem("LOOPBACK_SERVER_ONLY", 500)
	}
	if len(token) < 32 || strings.ContainsAny(token, "\r\n") {
		return nil, problem("API_CREDENTIAL_MISSING", 500)
	}
	return &http.Server{Addr: e.cfg.Server.Listen, Handler: e.Handler(token), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: time.Duration(e.cfg.TimeoutSeconds+5) * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}, nil
}
