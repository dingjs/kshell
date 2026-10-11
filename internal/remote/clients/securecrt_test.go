package clients

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseSecureCRTSession(t *testing.T) {
	path := filepath.Join("testdata", "sample_securecrt.ini")
	cand, ok := parseSecureCRTSession(path)
	if !ok {
		t.Fatal("parse failed")
	}
	if cand.Host != "crt.example.net" || cand.User != "ops" || cand.Port != 2200 {
		t.Fatalf("fields: %+v", cand)
	}
	if !cand.NeedsPassword {
		t.Fatal("expected NeedsPassword for Password V2")
	}
	if cand.Source != "securecrt" {
		t.Fatalf("source = %q", cand.Source)
	}
}

func TestSecureCRTScannerScanWithDir(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "sample_securecrt.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "srv.ini"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		got, err := SecureCRTScanner{SessionsDir: dir}.Scan()
		if err != nil || got != nil {
			t.Fatalf("non-windows: err=%v got=%+v", err, got)
		}
		return
	}
	got, err := SecureCRTScanner{SessionsDir: dir}.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
}
