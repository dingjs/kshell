package clients

import (
	"os"
	"path/filepath"

	"github.com/yangk/kshell/internal/remote/scanners"
)

// SSHConfigScanner 解析用户 OpenSSH 配置（默认 ~/.ssh/config）。
type SSHConfigScanner struct {
	// ConfigPath 可注入以便测试；为空时使用 ~/.ssh/config。
	ConfigPath string
}

func (SSHConfigScanner) ID() string { return "sshconfig" }

func (s SSHConfigScanner) Scan() ([]ClientCandidate, error) {
	path := s.ConfigPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".ssh", "config")
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	raw := scanners.ParseSSHConfig(f)
	out := make([]ClientCandidate, 0, len(raw))
	for _, c := range raw {
		out = append(out, ClientCandidate{
			Name:         c.Name,
			Host:         c.Host,
			User:         c.User,
			Port:         c.Port,
			IdentityFile: c.IdentityFile,
			Source:       "sshconfig",
			SourceFile:   path,
			Confidence:   c.Confidence,
		})
	}
	return out, nil
}
