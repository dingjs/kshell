package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangk/kshell/internal/mcparchive"
	"github.com/yangk/kshell/internal/remote"
)

func TestMCPServerDefsMergeArchiveAndSSH(t *testing.T) {
	ws := filepath.Clean(`D:\ws-a`)
	store := remote.NewStore(filepath.Join(t.TempDir(), "connections.yaml"))
	if _, err := store.Add(remote.Connection{
		ID: "c1", Host: "10.0.0.1", Workspace: ws, AllowAgent: true,
	}); err != nil {
		t.Fatal(err)
	}
	app := NewAppWith(Options{Store: store})
	app.archiveHub = &mcparchive.Hub{URL: "http://127.0.0.1:1", Token: "tok-a"}
	app.sshHub = &SSHHub{URL: "http://127.0.0.1:2", tokens: map[string]string{}}

	defs, ok := app.mcpServerDefs(ws, "terminal:t1")
	if !ok || len(defs) != 2 {
		t.Fatalf("ok=%v len=%d defs=%+v", ok, len(defs), defs)
	}
	if defs[0].Name != "kshell" || defs[1].Name != "kshell-ssh" {
		t.Fatalf("names=%q %q", defs[0].Name, defs[1].Name)
	}
	if !strings.Contains(strings.Join(defs[1].Args, " "), "--hub-url") {
		t.Fatalf("ssh args=%v", defs[1].Args)
	}
}

func TestWriteMergedMCPJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	err := mcparchive.WriteServers(path, []mcparchive.ServerDef{
		{Name: "kshell", Command: `C:\kshell.exe`, Args: []string{"mcp-archive", "--url", "http://u", "--token", "a"}},
		{Name: "kshell-ssh", Command: `C:\kshell.exe`, Args: []string{"mcp-ssh", "--hub-url", "http://h", "--token", "b", "--workspace", `D:\ws`}},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Servers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Servers) != 2 || doc.Servers["kshell"].Command == "" || doc.Servers["kshell-ssh"].Args[0] != "mcp-ssh" {
		t.Fatalf("doc=%+v", doc.Servers)
	}
}

func TestMCPServerDefsSkipsSSHWithoutAllowAgent(t *testing.T) {
	ws := filepath.Clean(`D:\ws-a`)
	store := remote.NewStore(filepath.Join(t.TempDir(), "connections.yaml"))
	if _, err := store.Add(remote.Connection{ID: "c1", Host: "h", Workspace: ws}); err != nil {
		t.Fatal(err)
	}
	app := NewAppWith(Options{Store: store})
	app.sshHub = &SSHHub{URL: "http://127.0.0.1:2", tokens: map[string]string{}}
	defs, ok := app.mcpServerDefs(ws, "")
	if ok || len(defs) != 0 {
		t.Fatalf("ok=%v defs=%+v", ok, defs)
	}
}
