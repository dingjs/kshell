//go:build windows

package desktop

import (
	"runtime"
	"testing"

	"golang.org/x/sys/windows"
)

// 气泡消息循环必须与 OS 线程绑定：goroutine 被调度器迁移后 GetMessage 泵不到
// 创建窗口线程的队列，气泡会「无响应」。此处锁定可观测契约——钉住后线程 ID
// 非零、与钉之前一致，且跨调度点保持稳定（锁定后的保证行为）。
// 注意：Go 1.27 起 runtime.LockedOSThread 已从公开 API 移除，无法直接查询
// 锁定状态；真实迁移场景留冒烟验证，不在单测里复现。
func TestPinBubbleUIThread(t *testing.T) {
	before := windows.GetCurrentThreadId()
	pinned := pinBubbleUIThread()
	defer runtime.UnlockOSThread()
	if pinned == 0 {
		t.Fatalf("pinBubbleUIThread 应返回非零线程 ID，得到 0")
	}
	if pinned != before {
		t.Fatalf("钉住后线程 ID 不应变化：钉前 %d，钉后 %d", before, pinned)
	}
	for i := 0; i < 200; i++ {
		runtime.Gosched()
		if id := windows.GetCurrentThreadId(); id != pinned {
			t.Fatalf("已钉住的 goroutine 不应被调度器迁移：%d -> %d", pinned, id)
		}
	}
}
