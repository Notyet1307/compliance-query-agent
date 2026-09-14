package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func validateDaemonOrigin(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || raw != u.Scheme+"://"+u.Host {
		return problem("MANAGED_DAEMON_INVALID", 500)
	}
	ip := net.ParseIP(u.Hostname())
	// Plaintext is limited to the dedicated trial daemon or literal private/loopback IPs.
	// A live execution still requires explicit approval of this origin and network.
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "cqa-s2-02-daemon" || ip != nil && (ip.IsLoopback() || ip.IsPrivate()))) {
		return problem("MANAGED_DAEMON_INVALID", 500)
	}
	return nil
}

func managedGeneration(c Config) (GenerationConfig, error) {
	g := c.Generation
	raw := os.Getenv("OPENAI_BASE_URL")
	u, err := url.Parse(raw)
	if err != nil {
		return g, problem("MANAGED_MODEL_ROUTE_INVALID", 500)
	}
	parts := strings.Split(u.Path, "/")
	if len(parts) != 8 || parts[1] != "api" || parts[2] != "runtime" || parts[3] != "sandboxes" || parts[5] != "llm" || parts[6] != "openai" || parts[7] != "v1" {
		return g, problem("MANAGED_MODEL_ROUTE_INVALID", 500)
	}
	id, err := hex.DecodeString(parts[4])
	if err != nil || len(id) != 32 || raw != c.Managed.DaemonOrigin+"/api/runtime/sandboxes/"+parts[4]+"/llm/openai/v1" {
		return g, problem("MANAGED_MODEL_ROUTE_INVALID", 500)
	}
	g.Endpoint = raw + "/chat/completions"
	return g, nil
}

// No MkdirAll: a missing bind must not turn into a fresh ephemeral receipt store.
// mountInfo is the kernel mount table; accepting a supplied fixture is only used by offline tests.
func managedStore(dir string, mountInfo []byte) (*Store, error) {
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil || resolved != dir {
		return nil, problem("PERSISTENT_STORE_REQUIRED", 500)
	}
	st, err := os.Lstat(dir)
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		return nil, problem("STORE_PERMISSIONS_UNSAFE", 500)
	}
	unescape := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	for _, line := range strings.Split(string(mountInfo), "\n") {
		mount, super, ok := strings.Cut(line, " - ")
		fields, filesystem := strings.Fields(mount), strings.Fields(super)
		if !ok || len(fields) < 6 || len(filesystem) < 3 || unescape.Replace(fields[4]) != dir {
			continue
		}
		if filesystem[0] == "tmpfs" || filesystem[0] == "ramfs" || filesystem[0] == "overlay" ||
			!strings.Contains(","+fields[5]+",", ",rw,") || !strings.Contains(","+filesystem[2]+",", ",rw,") {
			return nil, problem("PERSISTENT_STORE_REQUIRED", 500)
		}
		return &Store{dir: dir, requireResultDigest: true, rootInfo: st}, nil
	}
	return nil, problem("PERSISTENT_STORE_REQUIRED", 500)
}

func executableDigest() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", problem("BUILD_IDENTITY_UNAVAILABLE", 500)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", problem("BUILD_IDENTITY_UNAVAILABLE", 500)
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", problem("BUILD_IDENTITY_UNAVAILABLE", 500)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
