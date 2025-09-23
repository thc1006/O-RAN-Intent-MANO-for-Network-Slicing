// +build !windows

package vxlan

import (
	"os/exec"
	"syscall"
)

// setPlatformSysProcAttr sets Unix-specific process attributes
func setPlatformSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}