// +build windows

package vxlan

import (
	"os/exec"
	"syscall"
)

// setPlatformSysProcAttr sets Windows-specific process attributes
func setPlatformSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}