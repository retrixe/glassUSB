//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func IsExFATAvailable() bool {
	_, err := exec.LookPath("mkfs.exfat")
	return err == nil
}

func MakeExFAT(device string) error {
	if out, err := exec.Command("mkfs.exfat", device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create exFAT filesystem: %w\noutput: %s", err, out)
	}
	return nil
}
