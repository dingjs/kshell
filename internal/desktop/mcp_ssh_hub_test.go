package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangk/kshell/internal/remote"
)

func TestSSHHubRejectsMissingToken(t *testing.T) {
	h, err := startSSHHub(sshHubDeps{
		allowAgent: func(string) []remote.Connection { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)

	res, err := http.Post(h.URL+"/ssh/list", "application/json", strings.NewReader(`{"workspace":"D:\\ws"}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", res.StatusCode)
	}
}

func TestSSHHubExecRejectsNonAllowAgent(t *testing.T) {
	ws := filepath.Clean(`D:\proj\a`)
	allowed := remote.Connection{ID: "ok", Host: "10.0.0.1", Workspace: ws, AllowAgent: true, Password: "secret"}
	all := []remote.Connection{
		allowed,
		{ID: "no", Host: "10.0.0.2", Workspace: ws, AllowAgent: false},
	}
	h, err := startSSHHub(sshHubDeps{
		allowAgent: func(wsPath string) []remote.Connection {
			return remote.AllowAgentConnections(all, wsPath)
		},
		connByID: func(id string) (remote.Connection, bool) {
			for _, c := range all {
				if c.ID == id {
					return c, true
				}
			}
			return remote.Connection{}, false
		},
		exec: func(context.Context, remote.Connection, string) (remote.Result, error) {
			t.Fatal("exec should not run")
			return remote.Result{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)

	tok, err := h.IssueScopedToken(ws)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"workspace":"` + strings.ReplaceAll(ws, `\`, `\\`) + `","connection_id":"no","command":"true"}`
	req, _ := http.NewRequest(http.MethodPost, h.URL+"/ssh/exec", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", res.StatusCode)
	}

	bodyOK := `{"workspace":"` + strings.ReplaceAll(ws, `\`, `\\`) + `","connection_id":"ok","command":"echo hi"}`
	var execCmd string
	h2, err := startSSHHub(sshHubDeps{
		allowAgent: func(wsPath string) []remote.Connection {
			return remote.AllowAgentConnections(all, wsPath)
		},
		connByID: func(id string) (remote.Connection, bool) {
			for _, c := range all {
				if c.ID == id {
					return c, true
				}
			}
			return remote.Connection{}, false
		},
		exec: func(_ context.Context, c remote.Connection, cmd string) (remote.Result, error) {
			execCmd = cmd
			if c.ID != "ok" {
				t.Fatalf("conn id=%q", c.ID)
			}
			return remote.Result{Stdout: "hi", ExitCode: 0}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h2.Close)
	tok2, err := h2.IssueScopedToken(ws)
	if err != nil {
		t.Fatal(err)
	}
	req2, _ := http.NewRequest(http.MethodPost, h2.URL+"/ssh/exec", strings.NewReader(bodyOK))
	req2.Header.Set("Authorization", "Bearer "+tok2)
	req2.Header.Set("Content-Type", "application/json")
	res2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK || execCmd != "echo hi" {
		t.Fatalf("status=%d cmd=%q", res2.StatusCode, execCmd)
	}
}

func TestSSHHubListOmitsPassword(t *testing.T) {
	ws := filepath.Clean(`D:\proj\a`)
	all := []remote.Connection{{
		ID: "ok", Host: "h", Workspace: ws, AllowAgent: true, Password: "s3cret",
	}}
	h, err := startSSHHub(sshHubDeps{
		allowAgent: func(wsPath string) []remote.Connection {
			return remote.AllowAgentConnections(all, wsPath)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)

	tok, err := h.IssueScopedToken(ws)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, h.URL+"/ssh/list", bytes.NewReader([]byte(`{"workspace":"`+strings.ReplaceAll(ws, `\`, `\\`)+`"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var payload struct {
		Connections []remote.Connection `json:"connections"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Connections) != 1 || payload.Connections[0].Password != "" {
		t.Fatalf("connections=%+v", payload.Connections)
	}
}

func TestSSHHubRejectsWorkspaceMismatch(t *testing.T) {
	wsA := filepath.Clean(`D:\proj\a`)
	wsB := filepath.Clean(`D:\proj\b`)
	h, err := startSSHHub(sshHubDeps{
		allowAgent: func(string) []remote.Connection { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)

	tok, err := h.IssueScopedToken(wsA)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"workspace":"` + strings.ReplaceAll(wsB, `\`, `\\`) + `"}`
	req, _ := http.NewRequest(http.MethodPost, h.URL+"/ssh/list", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.StatusCode)
	}
}
