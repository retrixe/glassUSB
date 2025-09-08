//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func IsNTFSAvailable() bool {
	_, err := exec.LookPath("mkfs.ntfs")
	return err == nil
}

func MakeNTFS(device string) error {
	if out, err := exec.Command("mkfs.ntfs", device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create NTFS filesystem: %w\noutput: %s", err, out)
	}
	return nil
}
