package mcpssh

import (
	"strings"
	"testing"
)

func TestInjectArgsClaude(t *testing.T) {
	got := InjectArgs("claude", `C:\mcp.json`, []string{"--resume", "s1"})
	want := []string{"--mcp-config", `C:\mcp.json`, "--resume", "s1"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("claude args = %#v", got)
	}
}

func TestInjectArgsUnsupportedToolsUnchanged(t *testing.T) {
	plain := []string{"resume", "s1"}
	path := `C:\mcp.json`
	for _, toolID := range []string{"codex", "gemini", "opencode", "cursor", ""} {
		got := InjectArgs(toolID, path, plain)
		if strings.Join(got, "|") != "resume|s1" {
			t.Fatalf("%q should be unchanged: %#v", toolID, got)
		}
	}
}

func TestInjectArgsEmptyConfigPath(t *testing.T) {
	plain := []string{"a", "b"}
	if strings.Join(InjectArgs("claude", "", plain), "|") != "a|b" {
		t.Fatal("empty config path should not inject")
	}
}
