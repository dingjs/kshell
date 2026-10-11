package desktop

import (
	"sync"
	"time"

	"github.com/yangk/kshell/internal/agenthook"
)

// notifyDedupWindow 是同一条通知的去重时间窗：claude 等待确认期间的
// Notification hook 会连续重发同一条消息，窗口内重复事件直接丢弃，
// 避免气泡与提示音在一秒内连刷。
const notifyDedupWindow = 2 * time.Second

// notifyDeduper 按 (termKey, event, summary) 记录最近一次放行时刻，
// 窗口内的重复 payload 判定为重复。并发安全：收件箱轮询与 OSC 命中
// 分属不同 goroutine，都可能同时到达 handleAgentNotify。
type notifyDeduper struct {
	mu     sync.Mutex
	seen   map[string]time.Time
	now    func() time.Time // 测试注入用
	window time.Duration
}

func newNotifyDeduper(window time.Duration) *notifyDeduper {
	return &notifyDeduper{seen: make(map[string]time.Time), now: time.Now, window: window}
}

// dedupeKey 用 \x00 拼接判定字段：三类字段都不可能含该字节，无需转义。
func dedupeKey(p agenthook.Payload) string {
	return p.TermKey + "\x00" + p.Event + "\x00" + p.Summary
}

// notifyDedupFor 返回本实例的去重器（懒创建）。挂实例而非包级：测试各自
// new App 天然隔离，不因共享状态在同一窗口期内互相拦截。
func (a *App) notifyDedupFor() *notifyDeduper {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.notifyDedup == nil {
		a.notifyDedup = newNotifyDeduper(notifyDedupWindow)
	}
	return a.notifyDedup
}

// allow 报告该 payload 是否应放行：窗口内重复返回 false，否则记录并返回 true。
// 每次调用顺手清理过期记录，表大小受窗口内事件数约束。
func (d *notifyDeduper) allow(p agenthook.Payload) bool {
	now := d.now()
	cutoff := now.Add(-d.window)
	d.mu.Lock()
	defer d.mu.Unlock()
	for k, ts := range d.seen {
		if ts.Before(cutoff) {
			delete(d.seen, k)
		}
	}
	key := dedupeKey(p)
	// 窗口按左闭右开 [t, t+window)：恰好到达窗界视为过期，重新放行
	if ts, ok := d.seen[key]; ok && ts.After(cutoff) {
		return false
	}
	d.seen[key] = now
	return true
}
