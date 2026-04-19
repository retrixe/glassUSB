//go:build linux || darwin

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func MountPartition(partition string, mountPoint string) error {
	// use syscall.Mount instead? not portable on macOS though, and this didn't work on Linux
	// if err := syscall.Mount(partition, mountPoint, "", 0, ""); err != nil {

	if out, err := exec.Command("mount", partition, mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount partition: %w\noutput: %s", err, out)
	}
	return nil
}

func UnmountPartition(mountPoint string) error {
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		return fmt.Errorf("failed to unmount partition: %w", err)
	}
	return nil
}
