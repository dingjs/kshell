package clients

import "strings"

// PasswordFromSourceFile 从客户端会话文件再解析一次密码（Scan 响应不含密码、Import 在服务端填密时用）。
func PasswordFromSourceFile(source, sourceFile string) string {
	sourceFile = strings.TrimSpace(sourceFile)
	if sourceFile == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "xshell":
		if c, ok := parseXshellSession(sourceFile); ok {
			return c.Password
		}
	case "securecrt":
		if c, ok := parseSecureCRTSession(sourceFile); ok {
			return c.Password
		}
	}
	return ""
}

// RedactPasswords 清空候选密码，供 Scan API 返回前端前调用。
func RedactPasswords(cands []ClientCandidate) []ClientCandidate {
	if len(cands) == 0 {
		return cands
	}
	out := make([]ClientCandidate, len(cands))
	for i, c := range cands {
		c.Password = ""
		out[i] = c
	}
	return out
}
