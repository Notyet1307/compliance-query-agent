package agent

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	SchemaVersion   string           `json:"schemaVersion"`
	StoreDir        string           `json:"storeDir"`
	AllowSynthetic  bool             `json:"allowSynthetic"`
	NetworkApproved bool             `json:"networkApproved"`
	TimeoutSeconds  int              `json:"timeoutSeconds"`
	Knowledge       KnowledgeConfig  `json:"knowledge"`
	Generation      GenerationConfig `json:"generation"`
	Server          ServerConfig     `json:"server"`
}
type KnowledgeConfig struct {
	Mode       string `json:"mode"`
	CorpusPath string `json:"corpusPath,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	TokenEnv   string `json:"tokenEnv,omitempty"`
}

const generationProvider = "baizhi-chat"
const generationModel = "grok-4.6"
const generationEndpoint = "https://ai-api-gateway.app.baizhi.cloud/api/openai/chat/completions"

type GenerationConfig struct {
	Mode     string `json:"mode"`
	Endpoint string `json:"endpoint,omitempty"`
	Model    string `json:"model,omitempty"`
	TokenEnv string `json:"tokenEnv,omitempty"`
}
type ServerConfig struct {
	Listen   string `json:"listen"`
	TokenEnv string `json:"tokenEnv"`
}

// Relative data paths are resolved against the config file, not the caller cwd.
func LoadConfig(path string) (Config, error) {
	var c Config
	b, e := os.ReadFile(path)
	if e != nil || len(b) > 65536 {
		return c, problem("CONFIG_UNREADABLE", 500)
	}
	if StrictJSON(b, &c) != nil {
		return c, problem("CONFIG_INVALID", 500)
	}
	base, e := filepath.Abs(filepath.Dir(path))
	if e != nil {
		return c, problem("CONFIG_INVALID", 500)
	}
	if c.StoreDir != "" && !filepath.IsAbs(c.StoreDir) {
		c.StoreDir = filepath.Join(base, c.StoreDir)
	}
	if c.Knowledge.CorpusPath != "" && !filepath.IsAbs(c.Knowledge.CorpusPath) {
		c.Knowledge.CorpusPath = filepath.Join(base, c.Knowledge.CorpusPath)
	}
	return c, c.Validate()
}
func validateEndpoint(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(raw, "\r\n") {
		return problem("ENDPOINT_INVALID", 500)
	}
	if u.Scheme == "https" {
		return nil
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme == "http" && ip != nil && ip.IsLoopback() {
		return nil
	}
	return problem("TLS_REQUIRED", 500)
}
func (c Config) Validate() error {
	if c.SchemaVersion != "cqa.config/v1" || c.StoreDir == "" || c.TimeoutSeconds < 1 || c.TimeoutSeconds > 120 {
		return problem("CONFIG_INVALID", 500)
	}
	if c.Knowledge.Mode != "local" && c.Knowledge.Mode != "octobus_connect" {
		return problem("KNOWLEDGE_MODE_UNSUPPORTED", 500)
	}
	if c.Generation.Mode != "extractive" && c.Generation.Mode != "llm" {
		return problem("GENERATION_MODE_UNSUPPORTED", 500)
	}
	if c.Knowledge.Mode == "local" && c.Knowledge.CorpusPath == "" {
		return problem("CORPUS_REQUIRED", 500)
	}
	if c.Knowledge.Mode == "octobus_connect" {
		if !c.NetworkApproved {
			return problem("NETWORK_NOT_APPROVED", 403)
		}
		if e := validateEndpoint(c.Knowledge.Endpoint); e != nil {
			return e
		}
		u, _ := url.Parse(c.Knowledge.Endpoint)
		// This path is this project's proposed service contract, not a built-in OctoBus capability.
		if !strings.HasPrefix(u.Path, "/capsets/") || !strings.Contains(u.Path, "/connect/") || !strings.HasSuffix(u.Path, "/compliance.v1.KnowledgeService/Search") {
			return problem("OCTOBUS_CONTRACT_PATH_INVALID", 500)
		}
		if c.Knowledge.TokenEnv == "" {
			return problem("OCTOBUS_TOKEN_REF_REQUIRED", 500)
		}
	}
	if c.Generation.Mode == "llm" {
		if !c.NetworkApproved {
			return problem("NETWORK_NOT_APPROVED", 403)
		}
		if e := validateEndpoint(c.Generation.Endpoint); e != nil {
			return e
		}
		if c.Generation.Model != generationModel || c.Generation.TokenEnv == "" {
			return problem("MODEL_CONFIG_INVALID", 500)
		}
		u, _ := url.Parse(c.Generation.Endpoint)
		ip := net.ParseIP(u.Hostname())
		local := u.Scheme == "http" && ip != nil && ip.IsLoopback()
		if !local && c.Generation.Endpoint != generationEndpoint {
			return problem("MODEL_ENDPOINT_NOT_APPROVED", 403)
		}
	}
	return nil
}
