package clients

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func platformWindows() bool {
	return runtime.GOOS == "windows"
}

func defaultXshellSessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	docs := filepath.Join(home, "Documents")
	matches, err := filepath.Glob(filepath.Join(docs, "NetSarang*", "Xshell", "Sessions"))
	if err != nil {
		return "", err
	}
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && info.IsDir() {
			return m, nil
		}
	}
	// 部分安装路径在 AppData
	local := os.Getenv("LOCALAPPDATA")
	if local != "" {
		matches, _ = filepath.Glob(filepath.Join(local, "NetSarang*", "Xshell", "Sessions"))
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && info.IsDir() {
				return m, nil
			}
		}
	}
	return "", nil
}

func defaultSecureCRTSessionsDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", nil
	}
	candidates := []string{
		filepath.Join(appData, "VanDyke", "Config", "Sessions"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}
	return "", nil
}

func defaultMobaXtermConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var paths []string
	docs := filepath.Join(home, "Documents", "MobaXterm")
	paths = append(paths,
		filepath.Join(docs, "MobaXterm.ini"),
		filepath.Join(docs, ".mxtsessions"),
	)
	if appData := os.Getenv("APPDATA"); appData != "" {
		paths = append(paths, filepath.Join(appData, "MobaXterm", "MobaXterm.ini"))
	}
	return paths
}

func mobaSessionFromBookmarkValue(raw string) (host, user string, port int, ok bool) {
	// 常见：ssh # user@host | 22 或 ssh # user@host | port | ...
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", 0, false
	}
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		return "", "", 0, false
	}
	left := strings.TrimSpace(parts[0])
	port = 22
	if len(parts) >= 2 {
		if p, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && p > 0 {
			port = p
		}
	}
	// ssh # user@host
	if idx := strings.Index(left, "#"); idx >= 0 {
		left = strings.TrimSpace(left[idx+1:])
	}
	if at := strings.LastIndex(left, "@"); at >= 0 {
		user = strings.TrimSpace(left[:at])
		host = strings.TrimSpace(left[at+1:])
	} else {
		host = left
	}
	if host == "" {
		return "", "", 0, false
	}
	return host, user, port, true
}
