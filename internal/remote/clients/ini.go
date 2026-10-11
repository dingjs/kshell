package clients

import (
	"bufio"
	"io"
	"strings"
)

// readINIKeys 读取 INI 式 key=value（忽略节名，键名不区分大小写）。
func readINIKeys(r io.Reader) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}
		key, val, ok := splitINIAssignment(line)
		if !ok {
			continue
		}
		out[strings.ToLower(key)] = val
	}
	return out
}

func splitINIAssignment(line string) (key, val string, ok bool) {
	i := strings.IndexAny(line, "=:")
	if i < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:i])
	val = strings.TrimSpace(line[i+1:])
	val = strings.Trim(val, `"`)
	return key, val, key != ""
}

// readSecureCRTKeys 解析 SecureCRT 会话 ini（S:"Key"=value / D:"Key"=num）。
func readSecureCRTKeys(r io.Reader) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		// S:"Hostname"=host.example.com
		if len(line) < 4 || line[1] != ':' {
			continue
		}
		typ := line[0]
		rest := line[2:]
		eq := strings.Index(rest, "=")
		if eq < 0 {
			continue
		}
		keyPart := strings.TrimSpace(rest[:eq])
		val := strings.TrimSpace(rest[eq+1:])
		keyPart = strings.Trim(keyPart, `"`)
		switch typ {
		case 'S', 's':
			out[strings.ToLower(keyPart)] = val
		case 'D', 'd':
			out[strings.ToLower(keyPart)] = val
		}
	}
	return out
}
