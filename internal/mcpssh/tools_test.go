package mcpssh

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yangk/kshell/internal/remote"
)

type fakeBackend struct {
	conns []remote.Connection
	exec  func(ctx context.Context, id, cmd string) (stdout, stderr string, code int, err error)
	read  func(ctx context.Context, id, path string, limit int64) ([]byte, error)
	write func(ctx context.Context, id, path string, data []byte) error
}

func (f *fakeBackend) List() []remote.Connection {
	return append([]remote.Connection(nil), f.conns...)
}

func (f *fakeBackend) Exec(ctx context.Context, id, cmd string) (string, string, int, error) {
	if f.exec != nil {
		return f.exec(ctx, id, cmd)
	}
	return "", "", -1, nil
}

func (f *fakeBackend) ReadFile(ctx context.Context, id, path string, limit int64) ([]byte, error) {
	if f.read != nil {
		return f.read(ctx, id, path, limit)
	}
	return nil, nil
}

func (f *fakeBackend) WriteFile(ctx context.Context, id, path string, data []byte) error {
	if f.write != nil {
		return f.write(ctx, id, path, data)
	}
	return nil
}

func TestToolsListConnectionsRedactsSecrets(t *testing.T) {
	b := &fakeBackend{conns: []remote.Connection{{
		ID: "c1", Name: "prod", Host: "10.0.0.1", User: "root",
		Password: "s3cret", IdentityFile: `C:\keys\id_ed25519`,
	}}}
	text, isErr, err := NewTools(b).Call(context.Background(), "list_connections", nil)
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	if strings.Contains(text, "s3cret") {
		t.Fatalf("password leaked: %s", text)
	}
	var payload struct {
		Connections []remote.Connection `json:"connections"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Connections) != 1 || payload.Connections[0].Password != "" {
		t.Fatalf("connections=%+v", payload.Connections)
	}
	if payload.Connections[0].IdentityFile != `C:\keys\id_ed25519` {
		t.Fatalf("identity path missing: %+v", payload.Connections[0])
	}
}

func TestToolsUnknownConnectionID(t *testing.T) {
	b := &fakeBackend{conns: []remote.Connection{{ID: "c1", Host: "h"}}}
	tools := NewTools(b)
	cases := []struct {
		tool string
		args json.RawMessage
	}{
		{"ssh_exec", json.RawMessage(`{"connection_id":"missing","command":"ls"}`)},
		{"ssh_read_file", json.RawMessage(`{"connection_id":"missing","path":"/etc/hosts"}`)},
		{"ssh_write_file", json.RawMessage(`{"connection_id":"missing","path":"/tmp/x","content":"hi"}`)},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			_, isErr, err := tools.Call(context.Background(), tc.tool, tc.args)
			if err == nil || !isErr {
				t.Fatalf("want tool error, err=%v isErr=%v", err, isErr)
			}
			if !strings.Contains(err.Error(), "err.ssh.agent_not_allowed") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestToolsSSHExec(t *testing.T) {
	b := &fakeBackend{
		conns: []remote.Connection{{ID: "c1", Host: "h"}},
		exec: func(_ context.Context, id, cmd string) (string, string, int, error) {
			if id != "c1" || cmd != "echo hi" {
				t.Fatalf("id=%q cmd=%q", id, cmd)
			}
			return "hi\n", "", 0, nil
		},
	}
	text, isErr, err := NewTools(b).Call(context.Background(), "ssh_exec", json.RawMessage(`{"connection_id":"c1","command":"echo hi"}`))
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	var out struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}
	if out.Stdout != "hi\n" || out.ExitCode != 0 {
		t.Fatalf("out=%+v", out)
	}
}

func TestHandleToolsListAndCall(t *testing.T) {
	b := &fakeBackend{conns: []remote.Connection{{ID: "c1", Host: "h", Name: "dev"}}}
	tools := NewTools(b)
	list, ok := Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`), tools)
	if !ok || !strings.Contains(string(list), `"list_connections"`) || !strings.Contains(string(list), `"ssh_exec"`) {
		t.Fatalf("list ok=%v body=%s", ok, list)
	}
	resp, ok := Handle([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_connections","arguments":{}}}`), tools)
	if !ok || strings.Contains(string(resp), `"error"`) {
		t.Fatalf("call failed: %s", resp)
	}
	if !strings.Contains(string(resp), "dev") {
		t.Fatalf("missing conn: %s", resp)
	}
}

func TestMainRequiresToken(t *testing.T) {
	if err := Main([]string{}); err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("err=%v", err)
	}
	if err := Main([]string{"--token", "  "}); err == nil {
		t.Fatal("expected error for blank token")
	}
}
