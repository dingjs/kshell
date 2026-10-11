package mcpssh

import (
	"context"

	"github.com/yangk/kshell/internal/remote"
)

// Backend 抽象 Agent MCP 可调用的 SSH 能力；桌面 Hub 在后续任务接入真实实现。
type Backend interface {
	List() []remote.Connection
	Exec(ctx context.Context, id, cmd string) (stdout, stderr string, code int, err error)
	ReadFile(ctx context.Context, id, path string, limit int64) ([]byte, error)
	WriteFile(ctx context.Context, id, path string, data []byte) error
}

// noopBackend 占位：Main 在 Hub 未接入前仍可启动 stdio 服务。
type noopBackend struct{}

func (noopBackend) List() []remote.Connection { return nil }

func (noopBackend) Exec(context.Context, string, string) (string, string, int, error) {
	return "", "", -1, errAgentNotAllowed
}

func (noopBackend) ReadFile(context.Context, string, string, int64) ([]byte, error) {
	return nil, errAgentNotAllowed
}

func (noopBackend) WriteFile(context.Context, string, string, []byte) error {
	return errAgentNotAllowed
}
