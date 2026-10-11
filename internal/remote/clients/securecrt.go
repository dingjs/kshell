package clients

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SecureCRTScanner 扫描 VanDyke SecureCRT 会话目录下的 *.ini。
type SecureCRTScanner struct {
	SessionsDir string
}

func (SecureCRTScanner) ID() string { return "securecrt" }

func (s SecureCRTScanner) Scan() ([]ClientCandidate, error) {
	if !platformWindows() {
		return nil, nil
	}
	root := s.SessionsDir
	if root == "" {
		var err error
		root, err = defaultSecureCRTSessionsDir()
		if err != nil || root == "" {
			return nil, nil
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []ClientCandidate
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".ini") {
			continue
		}
		path := filepath.Join(root, e.Name())
		cand, ok := parseSecureCRTSession(path)
		if !ok {
			continue
		}
		out = append(out, cand)
	}
	return out, nil
}

func parseSecureCRTSession(path string) (ClientCandidate, bool) {
	f, err := os.Open(path)
	if err != nil {
		return ClientCandidate{}, false
	}
	defer f.Close()

	keys := readSecureCRTKeys(f)
	host := firstNonEmpty(keys, "hostname")
	if host == "" {
		return ClientCandidate{}, false
	}
	user := firstNonEmpty(keys, "username")
	port := 22
	if p := firstNonEmpty(keys, "port"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			port = n
		}
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	cand := ClientCandidate{
		Name:         name,
		Host:         host,
		User:         user,
		Port:         port,
		IdentityFile: firstNonEmpty(keys, "identity filename v2", "identity filename", "public key path"),
		Source:       "securecrt",
		SourceFile:   path,
		Confidence:   "medium",
	}
	if pw := firstNonEmpty(keys, "password v2", "password"); pw != "" {
		cand.NeedsPassword = true
	}
	return cand, true
}
