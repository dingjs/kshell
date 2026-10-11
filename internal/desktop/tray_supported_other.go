//go:build !darwin

// 非 darwin 平台托盘行为保持不变（Windows 原生 Shell_NotifyIcon、Linux dbus，
// 均不依赖主线程 NSApp 事件循环）。
package desktop

// traySupportedPlatform 标记当前平台是否支持启动托盘消息循环。
// 用 var 而非 const：单测可临时替换验证两种分支。
var traySupportedPlatform = true
