package providers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yangk/kshell/internal/applang"
)

func TestFormatSessionMarkdownClaudeJSONL(t *testing.T) {
	// role 标题经 applang 直显：固定 en，避免全局语言态串扰。
	applang.SetForTest(t, "en")
	s := Session{
		ID:     "42a6304b-1fd3-45aa-b620-10aa37988f2a",
		ToolID: "claude",
		Title:  "修复上传白名单",
		Path:   "testdata/claude/basic.jsonl",
	}
	md, err := FormatSessionMarkdown(s)
	if err != nil {
		t.Fatalf("FormatSessionMarkdown: %v", err)
	}
	if !strings.Contains(md, "## User") || !strings.Contains(md, "## Assistant") {
		t.Fatalf("应含 User/Assistant 标题, got:\n%s", md)
	}
	if !strings.Contains(md, "修复上传白名单校验") {
		t.Fatalf("应含用户原文, got:\n%s", md)
	}
	if !strings.Contains(md, "FileStorageService") {
		t.Fatalf("应含助手原文, got:\n%s", md)
	}
}

// 会话预览必须全量读入：超过旧 512KB 上限的文件，末尾轮次不得丢失。
func TestFormatSessionMarkdownLargeFileFullyRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.jsonl")
	var b strings.Builder
	b.WriteString(`{"type":"user","message":{"role":"user","content":"开头的第一条消息"}}` + "\n")
	// 填充到远超旧 512KB 上限的噪音轮次。
	noise := strings.Repeat("占位内容。", 1024) // 约 5KB
	for i := 0; i < 120; i++ {
		b.WriteString(`{"type":"assistant","message":{"role":"assistant","content":"` + noise + `"}}` + "\n")
	}
	b.WriteString(`{"type":"assistant","message":{"role":"assistant","content":"结尾的最后一条助手消息"}}` + "\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() <= 512*1024 {
		t.Fatalf("fixture 应大于 512KB, size=%d err=%v", fi.Size(), err)
	}
	md, err := FormatSessionMarkdown(Session{ID: "big", ToolID: "claude", Path: path})
	if err != nil {
		t.Fatalf("FormatSessionMarkdown: %v", err)
	}
	if !strings.Contains(md, "开头的第一条消息") {
		t.Fatalf("应含首条消息, got len=%d", len(md))
	}
	if !strings.Contains(md, "结尾的最后一条助手消息") {
		t.Fatalf("超过 512KB 的末尾消息不得被截断, got len=%d", len(md))
	}
}

func TestFormatSessionMarkdownGeminiJSON(t *testing.T) {
	s := Session{ID: "gemini-7f3c91", ToolID: "gemini", Path: "testdata/gemini/session.json"}
	md, err := FormatSessionMarkdown(s)
	if err != nil {
		t.Fatalf("FormatSessionMarkdown: %v", err)
	}
	if !strings.Contains(md, "帮我看下这个构建脚本") || !strings.Contains(md, "好的") {
		t.Fatalf("gemini messages 未抽出, got:\n%s", md)
	}
}

func TestFormatSessionMarkdownOpencodeDB(t *testing.T) {
	// 正文经 applang 直显：固定 en，避免全局语言态串扰。
	applang.SetForTest(t, "en")
	s := Session{ID: "oc1", ToolID: "opencode", Title: "重构登录页", Path: `D:\fake\opencode.db`}
	md, err := FormatSessionMarkdown(s)
	if err != nil {
		t.Fatalf("sqlite 应返回说明文案而非读库错误: %v", err)
	}
	if !strings.Contains(md, "重构登录页") || !strings.Contains(md, "stored in a database") {
		t.Fatalf("应说明无法展开 sqlite 会话, got:\n%s", md)
	}
}

// 空 Path 是绑定错误（前端 toast），对外为 wire key。
func TestFormatSessionMarkdownNoPath(t *testing.T) {
	_, err := FormatSessionMarkdown(Session{ID: "x", ToolID: "claude"})
	if !errors.Is(err, errTranscriptNoPath) {
		t.Fatalf("空路径应报 errTranscriptNoPath: %v", err)
	}
	if err.Error() != "err.transcript.no_path" {
		t.Fatalf("空路径错误应为 wire key, got %q", err.Error())
	}
}

// 无法解析出轮次时走 fallback 正文（applang 直显，固定 en 断言）。
func TestFormatSessionMarkdownFallbackEmpty(t *testing.T) {
	applang.SetForTest(t, "en")
	path := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"progress"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	md, err := FormatSessionMarkdown(Session{ID: "x", ToolID: "claude", Path: path})
	if err != nil {
		t.Fatalf("FormatSessionMarkdown: %v", err)
	}
	if !strings.Contains(md, "No conversation content could be parsed") {
		t.Fatalf("fallback 正文未本地化, got:\n%s", md)
	}
}

func TestFormatSessionMarkdownMissingFile(t *testing.T) {
	s := Session{ID: "x", ToolID: "claude", Path: "testdata/claude/not-exists.jsonl"}
	if _, err := FormatSessionMarkdown(s); err == nil {
		t.Fatal("缺失文件应报错")
	}
}
