// +build windows

package vxlan

import (
	"os/exec"
	"syscall"
)

// setPlatformSysProcAttr sets Windows-specific process attributes
// TODO: This function is intended for Windows platform-specific usage
func setPlatformSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}