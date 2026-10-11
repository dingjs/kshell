package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangk/kshell/internal/remote"
	"github.com/yangk/kshell/internal/remote/clients"
)

func newImportApp(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	store := remote.NewStore(filepath.Join(t.TempDir(), "connections.yaml"))
	app := NewAppWith(Options{
		Home:  root,
		Store: store,
	})
	return app, root
}

func TestScanSSHClientConfigsStoreNotReady(t *testing.T) {
	app := NewApp()
	got := app.ScanSSHClientConfigs()
	if got == nil {
		t.Fatal("应返回非 nil 空切片")
	}
	if len(got) != 0 {
		t.Fatalf("Store 未就绪时期望空列表, got %d", len(got))
	}
}

func TestScanSSHClientConfigsWithStore(t *testing.T) {
	app, _ := newImportApp(t)
	got := app.ScanSSHClientConfigs()
	if got == nil {
		t.Fatal("应返回非 nil 切片")
	}
	for _, c := range got {
		if c.Password != "" {
			t.Fatalf("Scan 响应不得含 Password: %+v", c)
		}
	}
}

func TestScanSSHClientConfigsRedactsPassword(t *testing.T) {
	redacted := clients.RedactPasswords([]clients.ClientCandidate{
		{Host: "10.0.0.1", Password: "must-not-leak"},
	})
	if len(redacted) != 1 || redacted[0].Password != "" {
		t.Fatalf("RedactPasswords: %+v", redacted)
	}
}

func TestImportSSHConnectionsMissingWorkspace(t *testing.T) {
	app, _ := newImportApp(t)
	_, err := app.ImportSSHConnections("", []clients.ClientCandidate{
		{Host: "10.0.0.1", Source: "xshell"},
	})
	if !errors.Is(err, errMissingWorkspace) {
		t.Fatalf("err = %v, want errMissingWorkspace", err)
	}
}

func TestImportSSHConnectionsStoreNotReady(t *testing.T) {
	app := NewApp()
	_, err := app.ImportSSHConnections("D:\\ws", nil)
	if !errors.Is(err, errNotReady) {
		t.Fatalf("err = %v, want errNotReady", err)
	}
}

func TestImportSSHConnectionsAddsAndDedupes(t *testing.T) {
	app, ws := newImportApp(t)
	wsSlash := ws + string(os.PathSeparator)

	items := []clients.ClientCandidate{
		{
			Name: "srv-a", Host: "10.0.0.20", User: "root", Port: 22,
			IdentityFile: `C:\Users\me\.ssh\id_rsa`,
			Source: "xshell", SourceFile: `C:\xshell\sessions\a.xsh`,
		},
		{
			Host: "10.0.0.20", User: "root", Port: 22,
			Source: "putty",
		},
		{
			Host: "10.0.0.21", User: "dev", Port: 0,
			Password: "secret", Source: "mobaxterm",
		},
	}

	added, err := app.ImportSSHConnections(wsSlash, items)
	if err != nil {
		t.Fatalf("ImportSSHConnections: %v", err)
	}
	if len(added) != 2 {
		t.Fatalf("期望导入 2 条（重复跳过 1）, got %d: %+v", len(added), added)
	}

	byHost := map[string]remote.Connection{}
	for _, c := range added {
		byHost[c.Host] = c
	}
	a := byHost["10.0.0.20"]
	if a.Source != "xshell" || a.Workspace != filepath.Clean(ws) {
		t.Fatalf("第一条字段不符: %+v", a)
	}
	if a.IdentityFile != `C:\Users\me\.ssh\id_rsa` || a.AllowAgent {
		t.Fatalf("IdentityFile/AllowAgent: %+v", a)
	}
	if a.Password != "" {
		t.Fatalf("无密码候选不应写入 Password")
	}

	b := byHost["10.0.0.21"]
	if b.Password != "" || b.Port != 22 || b.AllowAgent {
		t.Fatalf("Import 忽略前端 Password（无 SourceFile）: %+v", b)
	}

	again, err := app.ImportSSHConnections(ws, items[:1])
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("重复 workspace+host+user+port 应全部跳过, got %+v", again)
	}

	all := app.ListConnections(ws)
	var count20 int
	for _, c := range all {
		if c.Host == "10.0.0.20" {
			count20++
		}
	}
	if count20 != 1 {
		t.Fatalf("存储中 10.0.0.20 应仅 1 条, got %d", count20)
	}
}

func TestImportSSHConnectionsAllAddFails(t *testing.T) {
	app, ws := newImportApp(t)
	_, err := app.ImportSSHConnections(ws, []clients.ClientCandidate{
		{
			Host:         "10.0.0.99",
			IdentityFile: "-----BEGIN PRIVATE KEY-----",
			Source:       "manual",
		},
	})
	if !errors.Is(err, errImportAllFailed) {
		t.Fatalf("err = %v, want errImportAllFailed", err)
	}
}

func TestImportSSHConnectionsSkipsEmptyHost(t *testing.T) {
	app, ws := newImportApp(t)
	added, err := app.ImportSSHConnections(ws, []clients.ClientCandidate{
		{Host: "  ", Source: "xshell"},
		{Host: "10.0.0.30", Source: "sshconfig"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0].Host != "10.0.0.30" {
		t.Fatalf("got %+v", added)
	}
}

func TestUpsertConnectionPersistsAllowAgent(t *testing.T) {
	app, ws := newImportApp(t)
	saved, err := app.UpsertConnection(remote.Connection{
		Host: "10.0.0.40", Workspace: ws, AllowAgent: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.AllowAgent {
		t.Fatal("UpsertConnection 应保留 AllowAgent=true")
	}
	raw, err := os.ReadFile(app.snapshot().Store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "allow_agent: true") {
		t.Fatalf("yaml 未持久化 allow_agent: %q", string(raw))
	}
	reloaded := remote.NewStore(app.snapshot().Store.Path())
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	all := reloaded.All()
	if len(all) != 1 || !all[0].AllowAgent {
		t.Fatalf("重载后 AllowAgent 丢失: %+v", all)
	}
}
