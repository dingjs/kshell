package sessionarchive

import (
	"path/filepath"
	"testing"
)

func TestStorePersistsAcrossOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "archived.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Add("s1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("s1"); err != nil {
		t.Fatal(err)
	}
	if !s.Has("s1") || len(s.IDs()) != 1 {
		t.Fatalf("ids = %#v", s.IDs())
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Has("s1") {
		t.Fatal("reopen lost s1")
	}
	if err := s2.Remove("s1"); err != nil {
		t.Fatal(err)
	}
	s3, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if s3.Has("s1") {
		t.Fatal("remove did not persist")
	}
}

// AddMany 去重、跳过空 ID，且只落盘一次。
func TestAddManyDedupesAndPersistsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archived.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.AddMany([]string{"a", "", "b", "a"})
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v, want 2 nil", n, err)
	}
	if got := s.IDs(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("IDs=%v, want [a b]", got)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Has("a") || !s2.Has("b") {
		t.Fatal("应已落盘")
	}
}

// RemoveMany 只统计实际存在的 ID，返回移除数。
func TestRemoveManyReturnsRemovedCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archived.json")
	s, _ := Open(path)
	_, _ = s.AddMany([]string{"a", "b"})
	n, err := s.RemoveMany([]string{"a", "x"})
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v, want 1 nil", n, err)
	}
	if s.Has("a") || !s.Has("b") {
		t.Fatal("名单状态不对")
	}
}
