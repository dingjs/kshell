package desktop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yangk/kshell/internal/discovery"
)

var (
	errNoGitRepo      = errors.New("err.files.no_git_repo")
	errGitIgnoreWrite = errors.New("err.files.gitignore_write_failed")
)

// AddGitIgnore 把工作区内选中条目追加进其所在 git 仓库根的 .gitignore：
// 从条目目录向上找最近的含 .git 目录（不越出工作区根，工作区根自身命中也算），
// 目录条目追加「相对路径/」，文件追加相对路径（\ 归一为 /）；
// 与已有行完全相同则幂等成功；成功后作废树缓存。
func (a *App) AddGitIgnore(wsPath, relPath string) error {
	clean := filepath.Clean(strings.TrimSpace(relPath))
	if clean == "." || clean == "" {
		return errInvalidName
	}
	kind, local, _, err := a.parseWSRef(wsPath)
	if err != nil {
		return err
	}
	if kind == discovery.KindSSH {
		return errRemoteFSOpUnsupported
	}
	wsPath = local
	// showAll=true：超集树，已被忽略的条目同样可加 ignore
	tree, err := a.treeFor(wsPath, true)
	if err != nil {
		return err
	}
	node, err := a.nodeAt(tree, wsPath, relPath)
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(node.Path)
	if err != nil {
		return err // 目标是失效 junction：拒绝
	}
	if !underPath(wsPath, resolved) {
		return errPathOutsideWorkspace // junction 解析后越出工作区：拒绝
	}

	repoRoot := nearestGitRoot(wsPath, resolved, node.IsDir)
	if repoRoot == "" {
		return errNoGitRepo
	}

	rel, err := filepath.Rel(repoRoot, resolved)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if node.IsDir {
		rel += "/"
	}
	if err := appendGitIgnoreLine(filepath.Join(repoRoot, ".gitignore"), rel); err != nil {
		return fmt.Errorf("%w|%v", errGitIgnoreWrite, err)
	}

	a.invalidateTree(wsPath)
	return nil
}

// nearestGitRoot 从条目路径向上找最近的含 .git 目录：
// 目录取自身、文件取父目录为起点；搜索范围含工作区根、不含其上层。
func nearestGitRoot(wsPath, entryPath string, isDir bool) string {
	cur := entryPath
	if !isDir {
		cur = filepath.Dir(entryPath)
	}
	for {
		if _, err := os.Lstat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		if samePath(cur, wsPath) {
			return "" // 工作区根是搜索上界，更上层不算
		}
		parent := filepath.Dir(cur)
		if samePath(parent, cur) {
			return ""
		}
		cur = parent
	}
}

// appendGitIgnoreLine 幂等追加一行：文件不存在则创建（UTF-8、LF）；
// 已有内容且末尾无换行先补换行；与已有行完全相同则不重复追加。
// 比较时容忍 CRLF 行尾（\r 不算行内容差异），新写入统一用 LF。
func appendGitIgnoreLine(path, line string) error {
	content := ""
	if raw, err := os.ReadFile(path); err == nil {
		content = string(raw)
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSuffix(l, "\r") == line {
			return nil
		}
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line + "\n"
	return os.WriteFile(path, []byte(content), 0o666)
}
