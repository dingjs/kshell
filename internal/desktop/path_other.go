//go:build !darwin

package desktop

// 非 macOS：不存在 GUI 启动 PATH 精简的问题，空操作。
func augmentPath() {}