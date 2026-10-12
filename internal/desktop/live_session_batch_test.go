package desktop

import (
	"testing"

	"github.com/yangk/kshell/internal/config"
)

// BatchArchiveSessions 去重并跳过空 ID，全程只广播一次 archive:changed。
func TestBatchArchiveSessionsFiltersAndEmitsOnce(t *testing.T) {
	var events []string
	app := NewAppWith(Options{
		Config: config.Default(),
		Layout: config.Layout{Root: t.TempDir()},
		Emit:   func(name string, _ ...any) { events = append(events, name) },
	})
	n, err := app.BatchArchiveSessions([]string{"s1", "", "s2", "s1"})
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v, want 2 nil", n, err)
	}
	if len(events) != 1 || events[0] != "archive:changed" {
		t.Fatalf("events=%v, want 单次 archive:changed", events)
	}
	if ids := app.ArchivedIDs(); len(ids) != 2 {
		t.Fatalf("ArchivedIDs=%v, want 2 个", ids)
	}
}

// BatchRestoreSessions 只统计实际存在的 ID，剩余名单保持正确。
func TestBatchRestoreSessions(t *testing.T) {
	app := NewAppWith(Options{
		Config: config.Default(),
		Layout: config.Layout{Root: t.TempDir()},
		Emit:   func(string, ...any) {},
	})
	_, _ = app.BatchArchiveSessions([]string{"s1", "s2"})
	n, err := app.BatchRestoreSessions([]string{"s1", "s9"})
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v, want 1 nil", n, err)
	}
	if got := app.ArchivedIDs(); len(got) != 1 || got[0] != "s2" {
		t.Fatalf("ArchivedIDs=%v, want [s2]", got)
	}
}
