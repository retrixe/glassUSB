//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func MountPartition(partition string, mountPoint string) error {
	// TODO: use syscall.Mount instead?
	if out, err := exec.Command("mount", partition, mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount partition: %w\noutput: %s", err, out)
	}
	return nil
}

func UnmountPartition(mountPoint string) error {
	// TODO: use syscall.Unmount instead?
	if out, err := exec.Command("umount", mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount partition: %w\noutput: %s", err, out)
	}
	return nil
}
