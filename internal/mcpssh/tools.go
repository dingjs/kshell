package mcpssh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/yangk/kshell/internal/remote"
)

var errAgentNotAllowed = errors.New("err.ssh.agent_not_allowed")

// Tools 把 MCP 工具名映射到 Backend。
type Tools struct {
	backend Backend
}

// NewTools 绑定 Backend。
func NewTools(backend Backend) *Tools {
	return &Tools{backend: backend}
}

// ToolDefs 返回 tools/list 条目。
func (t *Tools) ToolDefs() []map[string]any {
	return []map[string]any{
		listConnectionsTool(),
		sshExecTool(),
		sshReadFileTool(),
		sshWriteFileTool(),
	}
}

// Call 执行工具，返回 JSON 文本、是否 MCP isError、以及供日志用的 err。
func (t *Tools) Call(ctx context.Context, name string, rawArgs json.RawMessage) (text string, isError bool, err error) {
	switch name {
	case "list_connections":
		return t.listConnections()
	case "ssh_exec":
		return t.sshExec(ctx, rawArgs)
	case "ssh_read_file":
		return t.sshReadFile(ctx, rawArgs)
	case "ssh_write_file":
		return t.sshWriteFile(ctx, rawArgs)
	default:
		return "", false, fmt.Errorf("unknown tool %s", name)
	}
}

func (t *Tools) listConnections() (string, bool, error) {
	conns := t.backend.List()
	safe := make([]remote.Connection, 0, len(conns))
	for _, c := range conns {
		safe = append(safe, sanitizeConnection(c))
	}
	b, err := json.Marshal(map[string]any{"connections": safe})
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}

func sanitizeConnection(c remote.Connection) remote.Connection {
	c.Password = ""
	if strings.Contains(c.IdentityFile, "BEGIN") {
		c.IdentityFile = ""
	}
	return c
}

func (t *Tools) sshExec(ctx context.Context, raw json.RawMessage) (string, bool, error) {
	var args struct {
		ConnectionID string `json:"connection_id"`
		Command      string `json:"command"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", true, err
	}
	if err := t.requireConnection(args.ConnectionID); err != nil {
		return "", true, err
	}
	stdout, stderr, code, err := t.backend.Exec(ctx, args.ConnectionID, args.Command)
	if err != nil {
		return "", true, err
	}
	b, err := json.Marshal(map[string]any{
		"stdout":    stdout,
		"stderr":    stderr,
		"exit_code": code,
	})
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}

func (t *Tools) sshReadFile(ctx context.Context, raw json.RawMessage) (string, bool, error) {
	var args struct {
		ConnectionID string `json:"connection_id"`
		Path         string `json:"path"`
		Limit        int64  `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", true, err
	}
	if err := t.requireConnection(args.ConnectionID); err != nil {
		return "", true, err
	}
	data, err := t.backend.ReadFile(ctx, args.ConnectionID, args.Path, args.Limit)
	if err != nil {
		return "", true, err
	}
	b, err := json.Marshal(map[string]any{
		"content": base64.StdEncoding.EncodeToString(data),
		"size":    len(data),
	})
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}

func (t *Tools) sshWriteFile(ctx context.Context, raw json.RawMessage) (string, bool, error) {
	var args struct {
		ConnectionID string `json:"connection_id"`
		Path         string `json:"path"`
		Content      string `json:"content"`
		Encoding     string `json:"encoding,omitempty"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", true, err
	}
	if err := t.requireConnection(args.ConnectionID); err != nil {
		return "", true, err
	}
	var data []byte
	switch strings.ToLower(args.Encoding) {
	case "", "utf-8", "utf8":
		data = []byte(args.Content)
	case "base64":
		var err error
		data, err = base64.StdEncoding.DecodeString(args.Content)
		if err != nil {
			return "", true, err
		}
	default:
		return "", true, fmt.Errorf("unsupported encoding %q", args.Encoding)
	}
	if err := t.backend.WriteFile(ctx, args.ConnectionID, args.Path, data); err != nil {
		return "", true, err
	}
	b, err := json.Marshal(map[string]any{"ok": true, "bytes": len(data)})
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}

func (t *Tools) requireConnection(id string) error {
	for _, c := range t.backend.List() {
		if c.ID == id {
			return nil
		}
	}
	return errAgentNotAllowed
}

func listConnectionsTool() map[string]any {
	return map[string]any{
		"name":        "list_connections",
		"description": "列出当前 Agent 会话允许使用的 SSH 连接（不含密码与私钥材料）。",
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func sshExecTool() map[string]any {
	return map[string]any{
		"name":        "ssh_exec",
		"description": "在指定连接上执行远程 shell 命令。",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]string{"type": "string", "description": "list_connections 返回的 id"},
				"command":       map[string]string{"type": "string", "description": "远程命令"},
			},
			"required": []string{"connection_id", "command"},
		},
	}
}

func sshReadFileTool() map[string]any {
	return map[string]any{
		"name":        "ssh_read_file",
		"description": "读取远程文件（返回 base64 内容）。",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]string{"type": "string"},
				"path":          map[string]string{"type": "string"},
				"limit":         map[string]string{"type": "integer", "description": "最大读取字节数，0 表示默认上限"},
			},
			"required": []string{"connection_id", "path"},
		},
	}
}

func sshWriteFileTool() map[string]any {
	return map[string]any{
		"name":        "ssh_write_file",
		"description": "写入远程文件。",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]string{"type": "string"},
				"path":          map[string]string{"type": "string"},
				"content":       map[string]string{"type": "string"},
				"encoding":      map[string]string{"type": "string", "description": "utf-8（默认）或 base64"},
			},
			"required": []string{"connection_id", "path", "content"},
		},
	}
}
