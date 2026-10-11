package desktop

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/yangk/kshell/internal/remote"
	remotefs "github.com/yangk/kshell/internal/remote/fs"
	"github.com/yangk/kshell/internal/workspace"
)

// SSHHub 是本机 SSH MCP 代理：mcp-ssh 子进程经 Bearer 调此处，密码不离开桌面进程。
type SSHHub struct {
	URL   string
	Token string
	srv   *http.Server
}

// sshHubDeps 抽象 Hub 依赖，便于单测注入 fake。
type sshHubDeps struct {
	allowAgent func(ws string) []remote.Connection
	connByID   func(id string) (remote.Connection, bool)
	exec       func(ctx context.Context, c remote.Connection, cmd string) (remote.Result, error)
	readFile   func(ctx context.Context, c remote.Connection, path string, limit int64) ([]byte, error)
	writeFile  func(ctx context.Context, c remote.Connection, path string, data []byte) error
}

func (a *App) sshHubDeps() sshHubDeps {
	return sshHubDeps{
		allowAgent: a.allowAgentConnections,
		connByID:   a.connByID,
		exec: func(ctx context.Context, c remote.Connection, cmd string) (remote.Result, error) {
			return remote.Run(ctx, c, cmd, a.sshOptions())
		},
		readFile: func(ctx context.Context, c remote.Connection, path string, limit int64) ([]byte, error) {
			if limit <= 0 {
				limit = workspace.MaxEditBytes
			}
			fs := &remotefs.FS{Conn: c, Opts: a.sshOptions(), Run: a.remoteRunner()}
			// 首期无远端 root 绑定时用 / 作根，允许绝对路径（用户已显式 AllowAgent）。
			return fs.ReadFile(ctx, "/", path, limit)
		},
		writeFile: func(ctx context.Context, c remote.Connection, path string, data []byte) error {
			fs := &remotefs.FS{Conn: c, Opts: a.sshOptions(), Run: a.remoteRunner()}
			return fs.WriteFile(ctx, "/", path, data)
		},
	}
}

func (a *App) allowAgentConnections(wsPath string) []remote.Connection {
	store := a.snapshot().Store
	if store == nil {
		return nil
	}
	return remote.AllowAgentConnections(store.All(), wsPath)
}

// StartSSHHub 在 127.0.0.1 随机端口监听 SSH MCP API。
func StartSSHHub(app *App) (*SSHHub, error) {
	if app == nil {
		return nil, nil
	}
	return startSSHHub(app.sshHubDeps())
}

func startSSHHub(d sshHubDeps) (*SSHHub, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(buf)
	mux := http.NewServeMux()
	h := &SSHHub{Token: token}

	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		return true
	}

	mux.HandleFunc("/ssh/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !auth(w, r) {
			return
		}
		ws, err := hubWorkspace(r)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		conns := d.allowAgent(ws)
		safe := make([]remote.Connection, 0, len(conns))
		for _, c := range conns {
			safe = append(safe, sanitizeConnForAgent(c))
		}
		writeHubJSON(w, map[string]any{"connections": safe})
	})

	mux.HandleFunc("/ssh/exec", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !auth(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		var body struct {
			Workspace    string `json:"workspace"`
			ConnectionID string `json:"connection_id"`
			Command      string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		c, ok := d.resolveAgentConn(body.Workspace, body.ConnectionID)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		res, err := d.exec(r.Context(), c, body.Command)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeHubJSON(w, map[string]any{
			"stdout":    res.Stdout,
			"stderr":    res.Stderr,
			"exit_code": res.ExitCode,
		})
	})

	mux.HandleFunc("/ssh/read", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !auth(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		var body struct {
			Workspace    string `json:"workspace"`
			ConnectionID string `json:"connection_id"`
			Path         string `json:"path"`
			Limit        int64  `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		c, ok := d.resolveAgentConn(body.Workspace, body.ConnectionID)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		data, err := d.readFile(r.Context(), c, body.Path, body.Limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeHubJSON(w, map[string]any{
			"content": base64.StdEncoding.EncodeToString(data),
			"size":    len(data),
		})
	})

	mux.HandleFunc("/ssh/write", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !auth(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		var body struct {
			Workspace    string `json:"workspace"`
			ConnectionID string `json:"connection_id"`
			Path         string `json:"path"`
			Content      string `json:"content"`
			Encoding     string `json:"encoding"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		c, ok := d.resolveAgentConn(body.Workspace, body.ConnectionID)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		data := []byte(body.Content)
		if strings.EqualFold(body.Encoding, "base64") {
			// Hub 写路径由 mcp-ssh 工具层解码后再以 utf-8 提交；此处仅兜底。
			http.Error(w, "unsupported encoding", http.StatusBadRequest)
			return
		}
		if err := d.writeFile(r.Context(), c, body.Path, data); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeHubJSON(w, map[string]any{"ok": true, "bytes": len(data)})
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	h.URL = "http://" + ln.Addr().String()
	h.srv = &http.Server{Handler: mux}
	go func() { _ = h.srv.Serve(ln) }()
	return h, nil
}

func (d sshHubDeps) resolveAgentConn(ws, id string) (remote.Connection, bool) {
	if id == "" || strings.TrimSpace(ws) == "" {
		return remote.Connection{}, false
	}
	for _, c := range d.allowAgent(ws) {
		if c.ID == id {
			if full, ok := d.connByID(id); ok {
				return full, true
			}
			return c, true
		}
	}
	return remote.Connection{}, false
}

func hubWorkspace(r *http.Request) (string, error) {
	if ws := strings.TrimSpace(r.URL.Query().Get("workspace")); ws != "" {
		return ws, nil
	}
	if r.Method == http.MethodGet {
		return "", nil
	}
	var body struct {
		Workspace string `json:"workspace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if err == io.EOF {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(body.Workspace), nil
}

func sanitizeConnForAgent(c remote.Connection) remote.Connection {
	c.Password = ""
	if strings.Contains(c.IdentityFile, "BEGIN") {
		c.IdentityFile = ""
	}
	return c
}

func writeHubJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// Close 停止 Hub 监听。
func (h *SSHHub) Close() {
	if h != nil && h.srv != nil {
		_ = h.srv.Close()
	}
}
