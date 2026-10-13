//go:build darwin

package desktop

import (
	"os"
	"path/filepath"
	"strings"
)

// macUserBinDirs 是 macOS GUI 启动（Finder/Dock）下 PATH 缺失、但用户 shell 里几乎
// 一定存在的用户级可执行目录。桌面 App 拿不到 shell profile 注入的 PATH，这些目录
// 里的 agent（如 Homebrew 装的 claude/codex）会被 Detect 的 LookPath 漏掉，
// 一路兜底到 ConfigDirs 误判为「已损坏」。
var macUserBinDirs = []string{
	"/opt/homebrew/bin", // Apple Silicon Homebrew
	"/usr/local/bin",    // Intel Homebrew
	"~/.local/bin",
	"~/bin",
}

// augmentPath 把 macUserBinDirs 中「目录真实存在、且不在当前 PATH 里」的项前置进 PATH。
// 必须在任何 Detect/扫描之前调用（Startup 开头）；Setenv 对本进程生效，
// 内嵌终端等后续子进程也会继承到补齐后的 PATH。
func augmentPath() {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	cur := os.Getenv("PATH")
	adds := computePathAdditions(cur, home, macUserBinDirs)
	if len(adds) == 0 {
		return
	}
	parts := append(append([]string{}, adds...), filepath.SplitList(cur)...)
	os.Setenv("PATH", strings.Join(parts, string(filepath.ListSeparator)))
}

// computePathAdditions 返回 dirs 中「目录真实存在、且尚未在 cur 里」的项，保持声明顺序。
// 拆成纯函数便于无副作用测试；dirs 支持 ~/ 前缀（相对 home）与绝对路径。
func computePathAdditions(cur, home string, dirs []string) []string {
	seen := make(map[string]bool, len(filepath.SplitList(cur)))
	for _, p := range filepath.SplitList(cur) {
		seen[filepath.Clean(p)] = true
	}
	var out []string
	for _, d := range dirs {
		exp := d
		if strings.HasPrefix(exp, "~/") {
			exp = filepath.Join(home, d[2:])
		}
		fi, err := os.Stat(exp)
		if err != nil || !fi.IsDir() {
			continue
		}
		key := filepath.Clean(exp)
		if !seen[key] {
			seen[key] = true
			out = append(out, exp)
		}
	}
	return out
}