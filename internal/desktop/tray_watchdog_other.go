//go:build !windows

// 非 Windows 平台无托盘消息循环看门狗需求（Linux dbus / macOS 事件循环不同，
// 且 macOS 托盘当前整体跳过），仅保留空实现供 tray.go / app.go 无差别调用。
package desktop

// startTrayWatchdog 空实现。
func startTrayWatchdog() {}

// postTrayWMNull 空实现（KB135788 仅适用 Windows 托盘菜单）。
func postTrayWMNull() {}

// showMenuErrBenign 非 Windows 无 TrackPopupMenu 语义，不判良性。
func showMenuErrBenign(error) bool { return false }
