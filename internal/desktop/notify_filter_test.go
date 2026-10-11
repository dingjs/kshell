package desktop

import (
	"sync"
	"testing"

	"github.com/yangk/kshell/internal/agenthook"
	"github.com/yangk/kshell/internal/terminal"
)

// newFilterTestApp 构造带终端管理器（stub 后端）的通知测试 App，
// 并收集 emit 的事件名。终端经真实 Manager 打开，Prompted/Kind 状态与线上一致。
func newFilterTestApp(t *testing.T) (*App, *[]string) {
	t.Helper()
	var mu sync.Mutex
	names := []string{}
	a := NewAppWith(Options{
		Terminals: newTerminalManagerWith(func(string, ...any) {}, &stubTermBackend{}),
		Emit: func(name string, data ...any) {
			mu.Lock()
			defer mu.Unlock()
			names = append(names, name)
		},
	})
	return a, &names
}

// openFilterTerminal 打开一个内嵌终端；prompt 为真时先提交一行输入置位 Prompted。
func openFilterTerminal(t *testing.T, a *App, kind string, prompt bool) {
	t.Helper()
	m := a.terminals()
	info, err := m.Open("new:1", terminal.Info{
		Kind: kind, Workspace: "D:/ws", ToolID: "codebuddy",
	}, terminal.Spec{Path: "x"}, 80, 24)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if prompt {
		if err := m.Write(info.ID, []byte("修登录页\r")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

// 新建会话在用户发出首条输入之前，等待确认类通知（Notification / attention）
// 属就绪期噪音：不 emit、不弹气泡。其余情形（已输入、恢复会话、外部窗口
// key 查不到、完成/出错事件）一律放行，行为与现状一致。
func TestDispatchNotifyFiltersUnpromptedWaiting(t *testing.T) {
	cases := []struct {
		name     string
		openKind string // 空串表示不开终端（模拟外部窗口 key）
		prompt   bool
		event    string
		wantPass bool
	}{
		{"新建未输入 Notification 丢弃", terminal.KindNew, false, "Notification", false},
		{"新建未输入 attention 丢弃", terminal.KindNew, false, "attention", false},
		{"新建已输入 Notification 放行", terminal.KindNew, true, "Notification", true},
		{"恢复会话未输入 Notification 放行", terminal.KindSession, false, "Notification", true},
		{"key 查不到 Notification 放行", "", false, "Notification", true},
		{"新建未输入完成事件放行", terminal.KindNew, false, "Stop", true},
		{"新建未输入出错事件放行", terminal.KindNew, false, "error", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			a, names := newFilterTestApp(t)
			if c.openKind != "" {
				openFilterTerminal(t, a, c.openKind, c.prompt)
			}
			writeInboxFile(t, dir, "1.json", agenthook.Payload{
				Tool: "codebuddy", Event: c.event, TermKey: "new:1", Workspace: "D:/ws", Summary: "等待输入",
			})

			a.dispatchNotifyOnce(dir)

			got := len(*names)
			if c.wantPass && got != 1 {
				t.Fatalf("应放行一次 notify:agent，得到 %d 次", got)
			}
			if !c.wantPass && got != 0 {
				t.Fatalf("就绪期等待确认通知应被过滤，得到 %d 次: %v", got, *names)
			}
		})
	}
}
