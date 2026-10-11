package clients

import (
	"sort"
	"strings"
)

// ClientCandidate 是本机 SSH 客户端里发现的连接候选（需用户确认后再导入）。
type ClientCandidate struct {
	Name          string
	Host          string
	User          string
	Port          int
	IdentityFile  string
	Source        string
	SourceFile    string
	Password      string
	NeedsPassword bool
	Confidence    string // high | medium | low
}

// Scanner 扫描一种本机 SSH 客户端配置来源。
type Scanner interface {
	ID() string
	Scan() ([]ClientCandidate, error)
}

var confidenceRank = map[string]int{"high": 3, "medium": 2, "low": 1}

// ScanAll 运行全部扫描器并按 host|user|port 去重，保留置信度最高的一条。
func ScanAll(scanners []Scanner) []ClientCandidate {
	var all []ClientCandidate
	for _, s := range scanners {
		cands, err := s.Scan()
		if err != nil {
			continue
		}
		all = append(all, cands...)
	}
	return dedupe(all)
}

func dedupe(cands []ClientCandidate) []ClientCandidate {
	type key struct {
		host, user string
		port       int
	}

	best := map[key]ClientCandidate{}
	for _, c := range cands {
		if c.Host == "" {
			continue
		}
		port := c.Port
		if port == 0 {
			port = 22
		}
		k := key{strings.ToLower(c.Host), strings.ToLower(c.User), port}
		if prev, ok := best[k]; ok && confidenceRank[prev.Confidence] >= confidenceRank[c.Confidence] {
			continue
		}
		best[k] = c
	}

	out := make([]ClientCandidate, 0, len(best))
	for _, c := range best {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if confidenceRank[out[i].Confidence] != confidenceRank[out[j].Confidence] {
			return confidenceRank[out[i].Confidence] > confidenceRank[out[j].Confidence]
		}
		if out[i].Host != out[j].Host {
			return out[i].Host < out[j].Host
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// DefaultScanners 返回全部内置本机客户端扫描器。
func DefaultScanners() []Scanner {
	return []Scanner{
		SSHConfigScanner{},
		XshellScanner{},
		SecureCRTScanner{},
		MobaXtermScanner{},
		PuTTYScanner{},
	}
}
