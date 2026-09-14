package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// ponytail: X1 reuses the fixed guest's grpcurl; a permanent high-rate client would use in-process gRPC.
// The executable and proto are trusted image files, never request/config-selected commands.
type nativeKnowledge struct {
	target, token, capset, instance string
	executable                      string
	timeoutSeconds                  int
}

func newNativeKnowledge(c Config) (*nativeKnowledge, error) {
	target, token := os.Getenv("CAP_GRPC_TARGET"), os.Getenv("CAP_TOKEN")
	host, portText, err := net.SplitHostPort(target)
	port, portErr := strconv.Atoi(portText)
	if err != nil || portErr != nil || port < 1 || port > 65535 || (!safeID.MatchString(host) && net.ParseIP(host) == nil) {
		return nil, problem("NATIVE_GATEWAY_INVALID", 500)
	}
	if c.Managed != nil {
		daemon, _ := url.Parse(c.Managed.DaemonOrigin)
		if host != daemon.Hostname() {
			return nil, problem("NATIVE_GATEWAY_INVALID", 500)
		}
	}
	if len(token) < 16 || !safeID.MatchString(token) {
		return nil, problem("NATIVE_CREDENTIAL_MISSING", 500)
	}
	return &nativeKnowledge{target, token, c.Knowledge.Capset, c.Knowledge.Instance, "/usr/local/bin/grpcurl", c.TimeoutSeconds}, nil
}

// Each stream is bounded independently. Writer failure closes its copy pipe;
// the caller's context and WaitDelay also bound a process retaining pipe handles.
type nativeOutput struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *nativeOutput) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buffer.Len() {
		b.exceeded = true
		return 0, problem("UPSTREAM_RESPONSE_TOO_LARGE", 502)
	}
	return b.buffer.Write(p)
}

func (o *nativeKnowledge) Search(ctx context.Context, r Request) (Corpus, error) {
	body, err := json.Marshal(queryForKnowledge(r))
	if err != nil {
		return Corpus{}, problem("UPSTREAM_REQUEST_INVALID", 500)
	}
	cmd := exec.CommandContext(ctx, o.executable,
		"-plaintext", "-expand-headers", "-use-reflection=false",
		"-import-path", "/opt/cqa/protocol",
		"-proto", "/opt/cqa/protocol/compliance.proto",
		"-H", "x-capability-sandbox-token: ${CAP_TOKEN}",
		"-H", "x-octobus-capset: "+o.capset,
		"-H", "x-octobus-instance: "+o.instance,
		"-max-time", strconv.Itoa(o.timeoutSeconds), "-max-msg-sz", "4194304",
		"-d", "@", o.target, "compliance.v1.KnowledgeService/Search")
	// Do not inherit ambient proxies, tracing, provider keys or unrelated credentials.
	// Header expansion happens in grpcurl, so the credential value is absent from argv.
	cmd.Env = []string{"CAP_TOKEN=" + o.token, "PATH=/usr/local/bin:/usr/bin:/bin"}
	cmd.Stdin = bytes.NewReader(body)
	cmd.WaitDelay = time.Second
	stdout := nativeOutput{limit: 4 * 1024 * 1024}
	stderr := nativeOutput{limit: 64 * 1024}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if stdout.exceeded || stderr.exceeded {
		return Corpus{}, problem("UPSTREAM_RESPONSE_TOO_LARGE", 502)
	}
	if bytes.Contains(stdout.buffer.Bytes(), []byte(o.token)) || bytes.Contains(stderr.buffer.Bytes(), []byte(o.token)) {
		return Corpus{}, problem("CREDENTIAL_REFLECTION_BLOCKED", 502)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) || errors.Is(err, exec.ErrNotFound) {
			return Corpus{}, problem("NATIVE_TOOL_UNAVAILABLE", 500)
		}
		var exit *exec.ExitError
		if ctx.Err() == nil && errors.As(err, &exit) && (exit.ExitCode() == 64+7 || exit.ExitCode() == 64+16) {
			return Corpus{}, problem("UPSTREAM_REJECTED", 502)
		}
		// Includes transport/TLS mistakes, cancellation and ambiguous remote outcomes.
		return Corpus{}, problem("UPSTREAM_OUTCOME_UNKNOWN", 502)
	}
	corpus, err := decodeKnowledgeResponse(stdout.buffer.Bytes())
	if err != nil {
		return Corpus{}, err
	}
	if !corpus.Synthetic {
		return Corpus{}, problem("NATIVE_X1_SYNTHETIC_ONLY", 403)
	}
	return corpus, nil
}
