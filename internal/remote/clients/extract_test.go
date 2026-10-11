package clients

import "testing"

func TestRedactPasswordsClearsPassword(t *testing.T) {
	in := []ClientCandidate{
		{Host: "h", Password: "secret", NeedsPassword: false},
		{Host: "h2", Password: "", NeedsPassword: true},
	}
	out := RedactPasswords(in)
	if len(out) != 2 || out[0].Password != "" || out[1].Password != "" {
		t.Fatalf("redact: %+v", out)
	}
	if !out[1].NeedsPassword {
		t.Fatal("NeedsPassword 应保留")
	}
}
