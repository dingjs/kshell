package clients

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseXshellSession(t *testing.T) {
	path := filepath.Join("testdata", "sample.xsh")
	cand, ok := parseXshellSession(path)
	if !ok {
		t.Fatal("parse failed")
	}
	if cand.Host != "lab.internal" || cand.User != "dev" || cand.Port != 2222 {
		t.Fatalf("fields: %+v", cand)
	}
	if cand.Name != "Lab VM" {
		t.Fatalf("name = %q", cand.Name)
	}
	if cand.IdentityFile == "" {
		t.Fatal("expected userkey")
	}
	if !cand.NeedsPassword || cand.Password != "" {
		t.Fatalf("password handling: NeedsPassword=%v Password=%q", cand.NeedsPassword, cand.Password)
	}
	if cand.Source != "xshell" {
		t.Fatalf("source = %q", cand.Source)
	}
}

func TestXshellScannerScanWithDir(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "sample.xsh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "demo.xsh"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		got, err := XshellScanner{SessionsDir: dir}.Scan()
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatalf("non-windows want nil, got %+v", got)
		}
		return
	}
	got, err := XshellScanner{SessionsDir: dir}.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Host != "lab.internal" {
		t.Fatalf("got %+v", got)
	}
}
