package remote

import "path/filepath"

// AllowAgentConnections 返回当前工作区下允许 Agent 使用的 SSH 连接副本。
// wsPath 为空时不返回任何连接；Workspace 为空的全局连接不匹配任何项目。
func AllowAgentConnections(conns []Connection, wsPath string) []Connection {
	if wsPath == "" {
		return nil
	}
	cleanWS := filepath.Clean(wsPath)
	out := make([]Connection, 0, len(conns))
	for _, c := range conns {
		if !c.AllowAgent {
			continue
		}
		if c.Workspace == "" {
			continue
		}
		if filepath.Clean(c.Workspace) != cleanWS {
			continue
		}
		out = append(out, c)
	}
	return out
}
