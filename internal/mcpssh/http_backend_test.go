package mcpssh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangk/kshell/internal/remote"
)

func TestHTTPBackendListAndExec(t *testing.T) {
	ws := filepath.Clean(`D:\ws-a`)
	token := "test-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/ssh/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"connections": []remote.Connection{{ID: "c1", Host: "h", Workspace: ws}},
			})
		case "/ssh/exec":
			var body struct {
				ConnectionID string `json:"connection_id"`
				Command      string `json:"command"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stdout": "ok", "stderr": "", "exit_code": 0,
			})
		case "/ssh/read":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content": base64.StdEncoding.EncodeToString([]byte("data")),
				"size":    4,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	b := &HTTPBackend{HubURL: srv.URL, Token: token, Workspace: ws}
	conns := b.List()
	if len(conns) != 1 || conns[0].ID != "c1" {
		t.Fatalf("conns=%+v", conns)
	}
	out, _, code, err := b.Exec(context.Background(), "c1", "true")
	if err != nil || code != 0 || out != "ok" {
		t.Fatalf("exec err=%v code=%d out=%q", err, code, out)
	}
	data, err := b.ReadFile(context.Background(), "c1", "/tmp/x", 1024)
	if err != nil || string(data) != "data" {
		t.Fatalf("read err=%v data=%q", err, data)
	}
}

func TestBackendFromFlagsHubRequiresWorkspace(t *testing.T) {
	if _, err := backendFromFlags([]string{"--token", "t", "--hub-url", "http://127.0.0.1:1"}); err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("err=%v", err)
	}
}
