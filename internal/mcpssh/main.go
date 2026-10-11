package mcpssh

import (
	"errors"
	"flag"
	"os"
	"strings"
)

// Main 是 `kshell mcp-ssh` 子命令：stdio MCP，Hub 鉴权在后续任务接入。
func Main(args []string) error {
	fs := flag.NewFlagSet("mcp-ssh", flag.ContinueOnError)
	token := fs.String("token", "", "bearer token from desktop hub")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*token) == "" {
		return errors.New("mcp-ssh: --token 必填")
	}
	_ = token // Hub 校验在后续任务使用
	tools := NewTools(noopBackend{})
	return ServeIO(os.Stdin, os.Stdout, tools)
}
