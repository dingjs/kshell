package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangk/kshell/internal/providers"
)

// 更新中形态的 provider：安装目录私有、NodeEntryScript 声明自更新形态，
// versions\ 下放新鲜 zip 模拟更新器下载完成待解压。
type updatingFakeProvider struct{ home string }

func (p *updatingFakeProvider) ID() string          { return "fake" }
func (p *updatingFakeProvider) DisplayName() string { return "Fake" }
func (p *updatingFakeProvider) DetectSpec(string) providers.DetectSpec {
	return providers.DetectSpec{
		BinName:         "no-such-bin-xyz",
		InstallDirs:     []string{filepath.Join(p.home, "fake-agent")},
		ConfigDirs:      []string{"~/.fake-tool"},
		NodeEntryScript: "index.js",
	}
}
func (p *updatingFakeProvider) SessionRoots(string) []string { return nil }
func (p *updatingFakeProvider) SessionFilePattern() string   { return "*.jsonl" }
func (p *updatingFakeProvider) ParseSession(string, []byte) (*providers.Session, error) {
	return nil, nil
}
func (p *updatingFakeProvider) NewSessionCmd(ws, bin string) providers.Launch {
	return providers.Launch{Path: bin, Dir: ws}
}
func (p *updatingFakeProvider) ResumeCmd(s providers.Session, bin string) providers.Launch {
	return providers.Launch{Path: bin, Dir: s.Workspace}
}

// Tool 必须透传 Detection.Updating，launch 层据此区分「更新中」与「残缺」。
func TestDetectAllPassesUpdatingThrough(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "fake-agent", "versions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "fake-agent", "versions", "fresh.zip"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".fake-tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := DetectAll(home, []providers.Provider{&updatingFakeProvider{home: home}})
	if len(tools) != 1 {
		t.Fatalf("len=%d, want 1", len(tools))
	}
	if !tools[0].Installed || !tools[0].Updating {
		t.Fatalf("Installed=%v Updating=%v, want true/true", tools[0].Installed, tools[0].Updating)
	}
	if tools[0].Broken {
		t.Fatal("更新中不得标记 Broken")
	}
	if tools[0].Source != "updating" {
		t.Fatalf("Source=%q, want updating", tools[0].Source)
	}
}
