package mcpssh

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yangk/kshell/internal/remote"
)

// HTTPBackend 经桌面本机 Hub 执行 SSH MCP，避免子进程直接读 connections.yaml。
type HTTPBackend struct {
	HubURL    string
	Token     string
	Workspace string
	Client    *http.Client
}

// List 拉取当前工作区 AllowAgent 白名单内的连接。
func (b *HTTPBackend) List() []remote.Connection {
	var out struct {
		Connections []remote.Connection `json:"connections"`
	}
	if err := b.postJSON("/ssh/list", map[string]string{"workspace": b.Workspace}, &out); err != nil {
		return nil
	}
	return out.Connections
}

// Exec 在指定连接上执行远程命令。
func (b *HTTPBackend) Exec(ctx context.Context, id, cmd string) (stdout, stderr string, code int, err error) {
	var out struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
	}
	body := map[string]any{
		"workspace":     b.Workspace,
		"connection_id": id,
		"command":       cmd,
	}
	if err := b.postJSONWithCtx(ctx, "/ssh/exec", body, &out); err != nil {
		return "", "", -1, err
	}
	return out.Stdout, out.Stderr, out.ExitCode, nil
}

// ReadFile 读取远端文件。
func (b *HTTPBackend) ReadFile(ctx context.Context, id, path string, limit int64) ([]byte, error) {
	var out struct {
		Content string `json:"content"`
		Size    int    `json:"size"`
	}
	body := map[string]any{
		"workspace":     b.Workspace,
		"connection_id": id,
		"path":          path,
		"limit":         limit,
	}
	if err := b.postJSONWithCtx(ctx, "/ssh/read", body, &out); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(out.Content)
}

// WriteFile 写入远端文件。
func (b *HTTPBackend) WriteFile(ctx context.Context, id, path string, data []byte) error {
	body := map[string]any{
		"workspace":     b.Workspace,
		"connection_id": id,
		"path":          path,
		"content":       string(data),
		"encoding":      "utf-8",
	}
	return b.postJSONWithCtx(ctx, "/ssh/write", body, nil)
}

func (b *HTTPBackend) postJSON(path string, body any, out any) error {
	return b.postJSONWithCtx(context.Background(), path, body, out)
}

func (b *HTTPBackend) postJSONWithCtx(ctx context.Context, path string, body any, out any) error {
	if b == nil || strings.TrimSpace(b.HubURL) == "" {
		return fmt.Errorf("mcp-ssh: hub url missing")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(b.HubURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.Token)
	req.Header.Set("Content-Type", "application/json")
	client := b.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("mcp-ssh: hub unauthorized")
	}
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("mcp-ssh: hub status %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	if out == nil {
		io.Copy(io.Discard, res.Body)
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
