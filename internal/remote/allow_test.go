package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllowAgentConnectionsEmptyWsPath(t *testing.T) {
	conns := []Connection{
		{ID: "1", Host: "h", AllowAgent: true, Workspace: "D:\\ws-a"},
	}
	got := AllowAgentConnections(conns, "")
	if len(got) != 0 {
		t.Fatalf("wsPath empty: got %d, want 0", len(got))
	}
}

func TestAllowAgentConnectionsRequiresAllowAgentTrue(t *testing.T) {
	ws := "D:\\ws-a"
	conns := []Connection{
		{ID: "1", Host: "h", AllowAgent: false, Workspace: ws},
		{ID: "2", Host: "h2", Workspace: ws},
	}
	got := AllowAgentConnections(conns, ws)
	if len(got) != 0 {
		t.Fatalf("got %d, want 0", len(got))
	}
}

func TestAllowAgentConnectionsGlobalWorkspaceExcluded(t *testing.T) {
	ws := "D:\\ws-a"
	conns := []Connection{
		{ID: "1", Host: "h", AllowAgent: true, Workspace: ""},
	}
	got := AllowAgentConnections(conns, ws)
	if len(got) != 0 {
		t.Fatalf("global connection must not match project: got %+v", got)
	}
}

func TestAllowAgentConnectionsWorkspaceMatch(t *testing.T) {
	ws := filepath.Clean("D:\\ws-a")
	conns := []Connection{
		{ID: "1", Host: "a", AllowAgent: true, Workspace: "D:\\ws-a"},
		{ID: "2", Host: "b", AllowAgent: true, Workspace: "D:\\ws-b"},
		{ID: "3", Host: "c", AllowAgent: true, Workspace: ws + "\\sub"},
	}
	got := AllowAgentConnections(conns, ws)
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("got %+v, want only id=1", got)
	}
}

func TestAllowAgentConnectionsUsesCleanPaths(t *testing.T) {
	ws := "D:\\ws-a\\"
	conns := []Connection{
		{ID: "1", Host: "a", AllowAgent: true, Workspace: "D:\\ws-a"},
	}
	got := AllowAgentConnections(conns, ws)
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("Clean path compare failed: got %+v", got)
	}
}

func TestAllowAgentRoundTripYAML(t *testing.T) {
	store := tempStore(t)
	_, err := store.Add(Connection{
		Name:       "agent",
		Host:       "10.0.0.1",
		AllowAgent: true,
		Workspace:  "D:\\ws-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "allow_agent: true") {
		t.Fatalf("yaml missing allow_agent: %q", string(raw))
	}
	reloaded := NewStore(store.Path())
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	all := reloaded.All()
	if len(all) != 1 || !all[0].AllowAgent {
		t.Fatalf("AllowAgent not preserved: %+v", all)
	}
}
