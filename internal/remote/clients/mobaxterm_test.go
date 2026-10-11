package clients

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseMobaXtermINI(t *testing.T) {
	path := filepath.Join("testdata", "sample_mobaxterm.ini")
	got := parseMobaXtermINI(string(mustReadFile(t, path)), path)
	if len(got) < 2 {
		t.Fatalf("want >=2, got %+v", got)
	}
	var lab, backup bool
	for _, c := range got {
		if c.Host == "lab.internal" && c.User == "dev" && c.Port == 2222 {
			lab = true
		}
		if c.Host == "backup.local" && c.User == "backup" {
			backup = true
		}
	}
	if !lab || !backup {
		t.Fatalf("missing entries: %+v", got)
	}
}

func TestParseMobaXtermXML(t *testing.T) {
	path := filepath.Join("testdata", "sample_mobaxterm.xml")
	data := mustReadFile(t, path)
	got := parseMobaXtermXML(data, path)
	if len(got) != 1 || got[0].Host != "xml.example.com" || got[0].User != "xmluser" || got[0].Port != 8022 {
		t.Fatalf("got %+v", got)
	}
}

func TestMobaXtermScannerScanPaths(t *testing.T) {
	ini := filepath.Join("testdata", "sample_mobaxterm.ini")
	if runtime.GOOS != "windows" {
		got, err := MobaXtermScanner{ConfigPaths: []string{ini}}.Scan()
		if err != nil || got != nil {
			t.Fatalf("non-windows: err=%v got=%+v", err, got)
		}
		return
	}
	got, err := MobaXtermScanner{ConfigPaths: []string{ini}}.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 {
		t.Fatalf("got %+v", got)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
