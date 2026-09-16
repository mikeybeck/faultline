//go:build windows

package notify

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	flashwAll       = 0x00000003
	flashwTimerNoFG = 0x0000000C
	mbIconWarning   = 0x00000030
)

type flashwinfo struct {
	size    uint32
	hwnd    syscall.Handle
	flags   uint32
	count   uint32
	timeout uint32
}

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procFlashWindowEx            = user32.NewProc("FlashWindowEx")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procMessageBeep              = user32.NewProc("MessageBeep")
	procGetConsoleWindow         = kernel32.NewProc("GetConsoleWindow")

	enumOnce     sync.Once
	enumCallback uintptr
	enumMu       sync.Mutex
	foundHwnd    uintptr
)

func sendWindows(title, body string, sound bool, onClick func()) error {
	_ = flashTaskbar(sound)
	ps := `
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Warning
$n.Visible = $true
$clicked = $false
$n.add_BalloonTipClicked({ $script:clicked = $true })
$n.ShowBalloonTip(10000, $env:FAULTLINE_TITLE, $env:FAULTLINE_BODY, [System.Windows.Forms.ToolTipIcon]::Warning)
$deadline = (Get-Date).AddSeconds(12)
while ((Get-Date) -lt $deadline) {
  [System.Windows.Forms.Application]::DoEvents()
  if ($clicked) { $n.Dispose(); exit 2 }
  Start-Sleep -Milliseconds 200
}
$n.Dispose()
exit 0
`
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-NonInteractive", "-Command", ps)
	cmd.Env = append(os.Environ(), "FAULTLINE_TITLE="+title, "FAULTLINE_BODY="+body)
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 2 {
		if onClick != nil {
			onClick()
		}
		return nil
	}
	_ = strings.TrimSpace(title)
	return nil
}

func flashTaskbar(sound bool) error {
	hwnd := findAppWindow()
	if hwnd != 0 {
		info := flashwinfo{
			size:  uint32(unsafe.Sizeof(flashwinfo{})),
			hwnd:  syscall.Handle(hwnd),
			flags: flashwAll | flashwTimerNoFG,
		}
		_, _, _ = procFlashWindowEx.Call(uintptr(unsafe.Pointer(&info)))
	}
	if sound {
		_, _, _ = procMessageBeep.Call(mbIconWarning)
	}
	return nil
}

func findAppWindow() uintptr {
	enumOnce.Do(func() {
		enumCallback = syscall.NewCallback(enumOwnWindow)
	})
	enumMu.Lock()
	defer enumMu.Unlock()
	foundHwnd = 0
	_, _, _ = procEnumWindows.Call(enumCallback, 0)
	if foundHwnd != 0 {
		return foundHwnd
	}
	hwnd, _, _ := procGetConsoleWindow.Call()
	return hwnd
}

func enumOwnWindow(hwnd, _ uintptr) uintptr {
	var pid uint32
	_, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid != uint32(os.Getpid()) {
		return 1
	}
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	if visible == 0 {
		return 1
	}
	foundHwnd = hwnd
	return 0
}
