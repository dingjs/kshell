package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// makeGitRepo 造一个「git 仓库」标记：仓库根定位只做 .git 存在性检查
//（与 workspace.isGitDir 同口径），无需真实 git。
func makeGitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readGitIgnore(t *testing.T, repoRoot string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestAddGitIgnoreFileAtRoot(t *testing.T) {
	env := newFilesEnv(t)
	makeGitRepo(t, env.root)

	if err := env.app.AddGitIgnore(env.root, "main.go"); err != nil {
		t.Fatalf("AddGitIgnore: %v", err)
	}
	if got := readGitIgnore(t, env.root); got != "main.go\n" {
		t.Fatalf(".gitignore = %q, want %q", got, "main.go\n")
	}
}

func TestAddGitIgnoreDirectoryTrailingSlash(t *testing.T) {
	env := newFilesEnv(t)
	makeGitRepo(t, env.root)

	if err := env.app.AddGitIgnore(env.root, "pkg"); err != nil {
		t.Fatalf("AddGitIgnore: %v", err)
	}
	if got := readGitIgnore(t, env.root); got != "pkg/\n" {
		t.Fatalf(".gitignore = %q, want %q（目录应带尾斜杠）", got, "pkg/\n")
	}
}

func TestAddGitIgnoreNestedRepoNearestWins(t *testing.T) {
	env := newFilesEnv(t)
	makeGitRepo(t, env.root) // 工作区根本身也是仓库
	writeFile(t, env.root, filepath.Join("sub", "inner.go"), "package sub\n")
	if err := os.Mkdir(filepath.Join(env.root, "sub", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	makeGitRepo(t, filepath.Join(env.root, "sub")) // 子目录是独立仓库，就近为准

	// 子仓库内文件：应写入 sub/.gitignore，相对 sub 的路径
	if err := env.app.AddGitIgnore(env.root, filepath.Join("sub", "inner.go")); err != nil {
		t.Fatalf("AddGitIgnore sub/inner.go: %v", err)
	}
	if got := readGitIgnore(t, filepath.Join(env.root, "sub")); got != "inner.go\n" {
		t.Fatalf("sub/.gitignore = %q, want %q", got, "inner.go\n")
	}
	// 根 .gitignore 不应被碰过（不存在即未被创建）
	if _, err := os.Stat(filepath.Join(env.root, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("根 .gitignore 不应被写入: %v", err)
	}

	// 子仓库内目录：相对路径 + 尾斜杠
	if err := env.app.AddGitIgnore(env.root, filepath.Join("sub", "pkg")); err != nil {
		t.Fatalf("AddGitIgnore sub/pkg: %v", err)
	}
	if got := readGitIgnore(t, filepath.Join(env.root, "sub")); got != "inner.go\npkg/\n" {
		t.Fatalf("sub/.gitignore = %q, want %q", got, "inner.go\npkg/\n")
	}

	// 根仓库条目仍写根 .gitignore
	if err := env.app.AddGitIgnore(env.root, "main.go"); err != nil {
		t.Fatalf("AddGitIgnore main.go: %v", err)
	}
	if got := readGitIgnore(t, env.root); got != "main.go\n" {
		t.Fatalf(".gitignore = %q, want %q", got, "main.go\n")
	}
}

func TestAddGitIgnoreIdempotent(t *testing.T) {
	env := newFilesEnv(t)
	makeGitRepo(t, env.root)
	writeFile(t, env.root, ".gitignore", "main.go\nbuild/\n")

	if err := env.app.AddGitIgnore(env.root, "main.go"); err != nil {
		t.Fatalf("AddGitIgnore: %v", err)
	}
	// 重复调用不重复追加，已有行原样保留
	if got := readGitIgnore(t, env.root); got != "main.go\nbuild/\n" {
		t.Fatalf(".gitignore = %q, want %q（应幂等）", got, "main.go\nbuild/\n")
	}
}

func TestAddGitIgnoreAppendNoTrailingNewline(t *testing.T) {
	env := newFilesEnv(t)
	makeGitRepo(t, env.root)
	writeFile(t, env.root, ".gitignore", "foo") // 末尾无换行

	if err := env.app.AddGitIgnore(env.root, "main.go"); err != nil {
		t.Fatalf("AddGitIgnore: %v", err)
	}
	if got := readGitIgnore(t, env.root); got != "foo\nmain.go\n" {
		t.Fatalf(".gitignore = %q, want %q（应先补换行）", got, "foo\nmain.go\n")
	}
}

func TestAddGitIgnoreNoRepo(t *testing.T) {
	env := newFilesEnv(t) // 无任何 .git

	err := env.app.AddGitIgnore(env.root, "main.go")
	if !errors.Is(err, errNoGitRepo) {
		t.Fatalf("无 git 仓库应报 errNoGitRepo，got %v", err)
	}
	if err.Error() != "err.files.no_git_repo" {
		t.Fatalf("wire key 不符: %q", err.Error())
	}
	// 不应误建 .gitignore
	if _, err := os.Stat(filepath.Join(env.root, ".gitignore")); !os.IsNotExist(err) {
		t.Fatal("失败时不应创建 .gitignore")
	}
}

func TestAddGitIgnoreEscapeRejected(t *testing.T) {
	env := newFilesEnv(t)
	makeGitRepo(t, env.root)

	if err := env.app.AddGitIgnore(env.root, "../outside"); !errors.Is(err, errPathOutsideWorkspace) {
		t.Fatalf("穿越应报 errPathOutsideWorkspace，got %v", err)
	}
	// 工作区根本身不可作为条目
	if err := env.app.AddGitIgnore(env.root, "."); !errors.Is(err, errInvalidName) {
		t.Fatalf("根条目应报 errInvalidName，got %v", err)
	}
}

func TestAddGitIgnoreEntryMissing(t *testing.T) {
	env := newFilesEnv(t)
	makeGitRepo(t, env.root)

	if err := env.app.AddGitIgnore(env.root, "nope.go"); !errors.Is(err, errDirNotFound) {
		t.Fatalf("条目不存在应报 errDirNotFound，got %v", err)
	}
}
