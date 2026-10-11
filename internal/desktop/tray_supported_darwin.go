// macOS 暂不支持托盘：energye/systray 的 Run 在非主线程跑 [NSApp run]，
// 与 Wails 独占主线程的 NSApp 事件循环冲突，触发 AppKit 断言（SIGTRAP）。
// external loop 模式（RunWithExternalLoop）的 start() 会抢走 Wails 的
// NSApp delegate 且缺 applicationShouldTerminateAfterLastWindowClosed，
// 会导致关窗即退出，故 macOS 先整体跳过托盘（2026-10-11 崩溃分析）。
package desktop

// traySupportedPlatform 标记当前平台是否支持启动托盘消息循环。
// 用 var 而非 const：单测可临时替换验证两种分支。
var traySupportedPlatform = false
