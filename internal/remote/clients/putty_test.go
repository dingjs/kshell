package clients

import (
	"runtime"
	"testing"
)

func TestParsePuTTYSession(t *testing.T) {
	fields := puTTYFieldValues{
		"HostName":    "putty.example.org",
		"PortNumber":  "2222",
		"UserName":    "puttyuser",
		"Protocol":    "ssh",
		"PublicKeyFile": "C:\\fake\\key.ppk",
	}
	cand, ok := parsePuTTYSession("My Server", fields)
	if !ok {
		t.Fatal("parse failed")
	}
	if cand.Host != "putty.example.org" || cand.User != "puttyuser" || cand.Port != 2222 {
		t.Fatalf("fields: %+v", cand)
	}
	if cand.Source != "putty" {
		t.Fatalf("source = %q", cand.Source)
	}
}

func TestParsePuTTYSessionSkipsNonSSH(t *testing.T) {
	_, ok := parsePuTTYSession("telnet", puTTYFieldValues{
		"HostName": "x",
		"Protocol": "telnet",
	})
	if ok {
		t.Fatal("expected skip non-ssh")
	}
}

func TestPuTTYScannerNonWindowsEmpty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("registry scan covered on windows CI/dev only")
	}
	got, err := PuTTYScanner{}.Scan()
	if err != nil || got != nil {
		t.Fatalf("err=%v got=%+v", err, got)
	}
}
