package process

import (
	"syscall"
	"unsafe"
)

func IsTerminal(fd uintptr) bool {
	var t syscall.Termios
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&t)))
	return err == 0
}
