package mcpssh

import "github.com/yangk/kshell/internal/mcparchive"

// InjectArgs 按工具能力把 MCP 配置路径注入启动参数；不支持则原样返回 args。
func InjectArgs(toolID, configPath string, args []string) []string {
	return mcparchive.PrependConfig(toolID, configPath, args)
}
