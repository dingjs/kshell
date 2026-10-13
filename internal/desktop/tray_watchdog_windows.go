//go:build windows

// 托盘看门狗（黑匣子诊断）。
//
// 背景：托盘右键无反应历史复发（3a8d9c9 前台兜底后仍复发）。本次排查已用实验证伪：
//   - goroutine 迁移导致消息队列错位（事故时循环初始存活，后续因迁移不该立即死）；
//   - AttachThreadInput 对不泵消息线程会阻塞（实验：立即返回 false）；
//   - 前台线程冻结会阻塞 SetForegroundWindow/TrackPopupMenu（实验：正常返回）；
//   - TrackPopupMenu 重入（库自带 "Popup menu already active" 防护）。
// 剩余假设需要死亡现场的 goroutine 栈才能定位，而死亡发生时托盘回调不会执行、
// 无法靠托盘自身日志取证，故从外部探测：看门狗 goroutine 每 2s 向托盘窗口
// SendMessageTimeout(WM_NULL)，连续失败即认定消息循环死亡，dump 全部 goroutine 栈。
package desktop

import (
	"log"
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	procGetClassNameW       = user32.NewProc("GetClassNameW")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
	procPostMessageW        = user32.NewProc("PostMessageW")
)

const (
	trayWatchdogProbeInterval = 2 * time.Second
	trayWatchdogProbeTimeout  = 1500 * time.Millisecond
	trayWatchdogDeathAfter    = 3            // 连续失败次数达到即判死
	trayWatchdogDumpInterval  = time.Minute  // 死亡持续时重 dump 的间隔
)

var trayWatchdogStarted atomic.Bool

// startTrayWatchdog 启动托盘消息循环看门狗（幂等；托盘未起则自动退出）。
func startTrayWatchdog() {
	if !trayWatchdogStarted.CompareAndSwap(false, true) {
		return
	}
	go func() {
		var hwnd uintptr
		for i := 0; i < 150; i++ { // 最多等 30s 托盘窗口出现
			if hwnd = findTrayHWND(); hwnd != 0 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if hwnd == 0 {
			log.Printf("tray-watchdog: 30s 内未找到托盘窗口，看门狗退出")
			return
		}
		log.Printf("tray-watchdog: 启动，托盘窗口=0x%X", hwnd)
		fails := 0
		var lastDump time.Time
		for {
			time.Sleep(trayWatchdogProbeInterval)
			ok, errno := probeTrayWindow(hwnd)
			if ok {
				if fails >= trayWatchdogDeathAfter {
					log.Printf("tray-watchdog: 消息循环恢复响应（此前连续失败 %d 次，疑似短暂阻塞而非死亡）", fails)
				}
				fails = 0
				continue
			}
			// 句柄失效（窗口销毁）≠ 循环死亡：托盘正常退出时窗口先销毁
			if errno == 1400 { // ERROR_INVALID_WINDOW_HANDLE
				if findTrayHWND() == 0 {
					log.Printf("tray-watchdog: 托盘窗口已消失（正常退出），看门狗退出")
					return
				}
				continue
			}
			fails++
			log.Printf("tray-watchdog: 探测失败 %d/%d（errno=%d）", fails, trayWatchdogDeathAfter, errno)
			if fails >= trayWatchdogDeathAfter && time.Since(lastDump) > trayWatchdogDumpInterval {
				lastDump = time.Now()
				buf := make([]byte, 4<<20)
				n := runtime.Stack(buf, true)
				log.Printf("tray-watchdog: 消息循环疑似死亡，dump 全部 goroutine 栈：\n%s", buf[:n])
			}
		}
	}()
}

// findTrayHWND 在本进程中找 SystrayClass 隐藏窗口（energye/systray 的托盘窗口）。
func findTrayHWND() uintptr {
	pid := uint32(os.Getpid())
	var found uintptr
	cb := syscall.NewCallback(func(h uintptr, lp uintptr) uintptr {
		var p uint32
		procGetWindowThreadProcID.Call(h, uintptr(unsafe.Pointer(&p)))
		if p != pid {
			return 1
		}
		buf := make([]uint16, 64)
		n, _, _ := procGetClassNameW.Call(h, uintptr(unsafe.Pointer(&buf[0])), 64)
		if syscall.UTF16ToString(buf[:n]) == "SystrayClass" {
			found = h
			return 0
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}

// probeTrayWindow 探测托盘窗口消息循环是否在泵队列。
// 返回 errno：0=成功；其余为 SendMessageTimeout/GetLastError 的错误码。
func probeTrayWindow(hwnd uintptr) (bool, syscall.Errno) {
	r, _, err := procSendMessageTimeoutW.Call(
		hwnd,
		0, // WM_NULL
		0, 0,
		2, // SMTO_NORMAL
		uintptr(trayWatchdogProbeTimeout.Milliseconds()),
		0,
	)
	if r != 0 {
		return true, 0
	}
	if e, ok := err.(syscall.Errno); ok {
		return false, e
	}
	return false, 0
}

// postTrayWMNull 向托盘窗口补发 WM_NULL（KB135788：TrackPopupMenu 返回后
// 必须补发，否则菜单 dismissed 后窗口残留菜单态，下一次右键被吞掉）。
func postTrayWMNull() {
	if hwnd := findTrayHWND(); hwnd != 0 {
		procPostMessageW.Call(hwnd, 0 /*WM_NULL*/, 0, 0)
	}
}

// showMenuErrBenign 判断 ShowMenu 错误是否良性：TrackPopupMenu 在用户点空白处
// 取消菜单时同样返回 0，此时 GetLastError 为 ERROR_SUCCESS(0)，不是真失败。
func showMenuErrBenign(err error) bool {
	if e, ok := err.(syscall.Errno); ok {
		return e == 0
	}
	return false
}
