//go:build windows

package platform

import (
	"fmt"
	"syscall"
	"unsafe"
)

var shellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

func OpenURL(rawURL string) error {
	operation, _ := syscall.UTF16PtrFromString("open")
	value, err := syscall.UTF16PtrFromString(rawURL)
	if err != nil {
		return err
	}
	result, _, callErr := shellExecuteW.Call(0, uintptr(unsafe.Pointer(operation)), uintptr(unsafe.Pointer(value)), 0, 0, 1)
	if result <= 32 {
		return fmt.Errorf("ShellExecuteW failed (%d): %w", result, callErr)
	}
	return nil
}

