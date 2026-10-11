package clients

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleSSHConfig = `
Host prod
    HostName prod.example.com
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_ed25519

Host staging
    HostName 10.0.0.5
    User ubuntu
`

func TestSSHConfigScannerScan(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")
	if err := os.WriteFile(cfgPath, []byte(sampleSSHConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := SSHConfigScanner{ConfigPath: cfgPath}.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2: %+v", len(got), got)
	}
	if got[0].Source != "sshconfig" || got[0].SourceFile != cfgPath {
		t.Fatalf("metadata wrong: %+v", got[0])
	}
	if got[0].Host != "prod.example.com" || got[0].User != "deploy" || got[0].Port != 2222 {
		t.Fatalf("prod wrong: %+v", got[0])
	}
	if !strings.Contains(got[0].IdentityFile, "id_ed25519") {
		t.Fatalf("identity: %+v", got[0])
	}
}

func TestSSHConfigScannerMissingFile(t *testing.T) {
	got, err := SSHConfigScanner{ConfigPath: filepath.Join(t.TempDir(), "missing")}.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("want nil, got %+v", got)
	}
}
