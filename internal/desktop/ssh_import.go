package desktop

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yangk/kshell/internal/remote"
	"github.com/yangk/kshell/internal/remote/clients"
)

var (
	errMissingWorkspace = errors.New("err.ssh.missing_workspace")
	errImportAllFailed  = errors.New("err.ssh.import_all_failed")
)

// ScanSSHClientConfigs 聚合本机 SSH 客户端配置候选；Store 未就绪时返回空切片。
// 响应不含 Password（NeedsPassword 仍保留）。
func (a *App) ScanSSHClientConfigs() []clients.ClientCandidate {
	if a.snapshot().Store == nil {
		return []clients.ClientCandidate{}
	}
	out := clients.ScanAll(clients.DefaultScanners())
	if out == nil {
		return []clients.ClientCandidate{}
	}
	return clients.RedactPasswords(out)
}

// ImportSSHConnections 将勾选的客户端候选写入连接存储，绑定到 wsPath 工作区。
// 同 workspace+host+user+port 已存在则跳过；AllowAgent 固定为 false。
// 忽略前端传来的 Password，有 SourceFile 时在服务端再解析一次。
func (a *App) ImportSSHConnections(wsPath string, items []clients.ClientCandidate) ([]remote.Connection, error) {
	store := a.snapshot().Store
	if store == nil {
		return nil, errNotReady
	}
	wsPath = strings.TrimSpace(wsPath)
	if wsPath == "" {
		return nil, errMissingWorkspace
	}
	ws := filepath.Clean(wsPath)

	var added []remote.Connection
	var addFailures int
	for _, item := range items {
		if strings.TrimSpace(item.Host) == "" {
			continue
		}
		port := item.Port
		if port <= 0 {
			port = 22
		}
		if hasConnInWorkspace(store, ws, item.Host, item.User, port) {
			continue
		}
		conn := remote.Connection{
			Name:         item.Name,
			Host:         item.Host,
			User:         item.User,
			Port:         port,
			IdentityFile: item.IdentityFile,
			Workspace:    ws,
			Source:       item.Source,
			SourceFile:   item.SourceFile,
			AllowAgent:   false,
		}
		if pw := clients.PasswordFromSourceFile(item.Source, item.SourceFile); pw != "" {
			conn.Password = pw
		}
		saved, err := store.Add(conn)
		if err != nil {
			addFailures++
			continue
		}
		added = append(added, saved)
	}
	if addFailures > 0 && len(added) == 0 {
		return nil, fmt.Errorf("%w|%d", errImportAllFailed, addFailures)
	}
	if added == nil {
		return []remote.Connection{}, nil
	}
	return added, nil
}

func hasConnInWorkspace(store *remote.Store, ws, host, user string, port int) bool {
	for _, c := range store.All() {
		cws := filepath.Clean(strings.TrimSpace(c.Workspace))
		if cws != ws {
			continue
		}
		p := c.Port
		if p <= 0 {
			p = 22
		}
		if strings.EqualFold(c.Host, host) && strings.EqualFold(c.User, user) && p == port {
			return true
		}
	}
	return false
}
