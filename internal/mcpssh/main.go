package mcpssh

import (
	"errors"
	"flag"
	"os"
	"strings"
)

// backendFromFlags 解析 mcp-ssh 子命令参数。
func backendFromFlags(args []string) (Backend, error) {
	fs := flag.NewFlagSet("mcp-ssh", flag.ContinueOnError)
	token := fs.String("token", "", "bearer token from desktop hub")
	hubURL := fs.String("hub-url", "", "desktop ssh hub base url")
	workspace := fs.String("workspace", "", "agent workspace path for allowlist")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if strings.TrimSpace(*token) == "" {
		return nil, errors.New("mcp-ssh: --token 必填")
	}
	if strings.TrimSpace(*hubURL) != "" {
		if strings.TrimSpace(*workspace) == "" {
			return nil, errors.New("mcp-ssh: --workspace 必填（与 --hub-url 合用）")
		}
		return &HTTPBackend{
			HubURL:    *hubURL,
			Token:     *token,
			Workspace: *workspace,
		}, nil
	}
	return noopBackend{}, nil
}

// Main 是 `kshell mcp-ssh` 子命令：stdio MCP，经桌面 Hub 执行 SSH 工具。
func Main(args []string) error {
	backend, err := backendFromFlags(args)
	if err != nil {
		return err
	}
	tools := NewTools(backend)
	return ServeIO(os.Stdin, os.Stdout, tools)
}
