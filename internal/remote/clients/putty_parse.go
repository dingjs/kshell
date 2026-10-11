package clients

import (
	"strconv"
	"strings"
)

// puTTYFieldValues 表示 PuTTY 注册表会话项（键名与注册表 value 名一致，大小写不敏感）。
type puTTYFieldValues map[string]string

func parsePuTTYSession(name string, fields puTTYFieldValues) (ClientCandidate, bool) {
	get := func(keys ...string) string {
		for _, k := range keys {
			for fk, fv := range fields {
				if strings.EqualFold(fk, k) {
					return fv
				}
			}
		}
		return ""
	}

	proto := strings.ToLower(get("Protocol"))
	if proto != "" && proto != "ssh" {
		return ClientCandidate{}, false
	}

	host := get("HostName")
	if host == "" {
		return ClientCandidate{}, false
	}
	port := 22
	if p := get("PortNumber"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			port = n
		}
	}
	user := get("UserName")
	keyFile := get("PublicKeyFile")

	return ClientCandidate{
		Name:         name,
		Host:         host,
		User:         user,
		Port:         port,
		IdentityFile: keyFile,
		Source:       "putty",
		SourceFile:   "registry:HKCU\\Software\\SimonTatham\\PuTTY\\Sessions",
		Confidence:   "medium",
	}, true
}
