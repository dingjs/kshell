package clients

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// XshellScanner 扫描 NetSarang Xshell 会话目录下的 *.xsh。
type XshellScanner struct {
	// SessionsDir 可注入；为空时在 Windows 下搜索 Documents\NetSarang*\Xshell\Sessions。
	SessionsDir string
}

func (XshellScanner) ID() string { return "xshell" }

func (s XshellScanner) Scan() ([]ClientCandidate, error) {
	if !platformWindows() {
		return nil, nil
	}
	root := s.SessionsDir
	if root == "" {
		var err error
		root, err = defaultXshellSessionsDir()
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
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".xsh") {
			continue
		}
		path := filepath.Join(root, e.Name())
		cand, ok := parseXshellSession(path)
		if !ok {
			continue
		}
		out = append(out, cand)
	}
	return out, nil
}

func parseXshellSession(path string) (ClientCandidate, bool) {
	f, err := os.Open(path)
	if err != nil {
		return ClientCandidate{}, false
	}
	defer f.Close()

	keys := readINIKeys(f)
	host := firstNonEmpty(keys, "host", "hostname")
	if host == "" {
		return ClientCandidate{}, false
	}
	user := firstNonEmpty(keys, "username", "user")
	port := 22
	if p := firstNonEmpty(keys, "port"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			port = n
		}
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if desc := keys["description"]; desc != "" {
		name = desc
	}

	cand := ClientCandidate{
		Name:         name,
		Host:         host,
		User:         user,
		Port:         port,
		IdentityFile: firstNonEmpty(keys, "userkey", "publickey", "identityfile"),
		Source:       "xshell",
		SourceFile:   path,
		Confidence:   "medium",
	}

	pw := firstNonEmpty(keys, "password")
	if pw != "" {
		if plain, ok := tryDecryptXshellPassword(pw); ok && plain != "" {
			cand.Password = plain
		} else {
			cand.NeedsPassword = true
		}
	}
	return cand, true
}

// tryDecryptXshellPassword 尽力解密；当前未实现可靠算法时返回 false。
func tryDecryptXshellPassword(_ string) (string, bool) {
	return "", false
}

func firstNonEmpty(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(m[strings.ToLower(k)]); v != "" {
			return v
		}
	}
	return ""
}
