//go:build windows

package googleapi

import (
	"syscall"
	"unsafe"
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// openBrowser opens the specified URL in the default browser on Windows.
// It uses ShellExecuteW directly instead of cmd.exe to avoid shell escaping issues (e.g. with ampersands).
func openBrowser(url string) error {
	urlPtr, err := syscall.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	opPtr, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	// ShellExecuteW(hwnd, lpOperation, lpFile, lpParameters, lpDirectory, nShowCmd)
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(opPtr)),
		uintptr(unsafe.Pointer(urlPtr)),
		0,
		0,
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		return syscall.Errno(ret)
	}
	return nil
}
