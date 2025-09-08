//go:build linux

package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func LoopMountFile(name string) (string, error) {
	out, err := exec.Command("losetup", "-f").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to find free loop device: %w\noutput: %s", err, out)
	}
	loopDevice := string(bytes.TrimSpace(out))
	if out, err = exec.Command("losetup", "-P", loopDevice, name).CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to set up loop device: %w\noutput: %s", err, out)
	}
	return loopDevice, nil
}

func LoopUnmountFile(loopDevice string) error {
	if out, err := exec.Command("losetup", "-d", loopDevice).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to detach loop device: %w\noutput: %s", err, out)
	}
	return nil
}

func MountPartition(partition string, mountPoint string) error {
	if out, err := exec.Command("mount", partition, mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount partition: %w\noutput: %s", err, out)
	}
	return nil
}

func UnmountPartition(mountPoint string) error {
	if out, err := exec.Command("umount", mountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount partition: %w\noutput: %s", err, out)
	}
	return nil
}
