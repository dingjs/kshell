package clients

import (
	"testing"
)

func TestScanAllDedupeByHostUserPort(t *testing.T) {
	stub := stubScanner{
		id: "a",
		cands: []ClientCandidate{
			{Host: "h1", User: "u", Port: 22, Confidence: "low", Source: "a", Name: "low"},
			{Host: "h1", User: "u", Port: 22, Confidence: "high", Source: "b", Name: "high"},
		},
	}
	got := ScanAll([]Scanner{stub})
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if got[0].Confidence != "high" || got[0].Name != "high" {
		t.Fatalf("kept wrong row: %+v", got[0])
	}
}

func TestScanAllSkipsScannerError(t *testing.T) {
	ok := stubScanner{id: "ok", cands: []ClientCandidate{{Host: "x", Port: 22, Confidence: "medium"}}}
	bad := stubScanner{id: "bad", err: errTest}
	got := ScanAll([]Scanner{bad, ok})
	if len(got) != 1 || got[0].Host != "x" {
		t.Fatalf("got %+v", got)
	}
}

func TestDefaultScannersIDs(t *testing.T) {
	want := map[string]bool{"sshconfig": true, "xshell": true, "securecrt": true, "mobaxterm": true, "putty": true}
	for _, s := range DefaultScanners() {
		if !want[s.ID()] {
			t.Fatalf("unexpected scanner id %q", s.ID())
		}
		delete(want, s.ID())
	}
	if len(want) != 0 {
		t.Fatalf("missing ids: %v", want)
	}
}

type stubScanner struct {
	id    string
	cands []ClientCandidate
	err   error
}

func (s stubScanner) ID() string { return s.id }

func (s stubScanner) Scan() ([]ClientCandidate, error) {
	return s.cands, s.err
}

var errTest = &testError{"scan failed"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
