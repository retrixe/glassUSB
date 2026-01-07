//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func IsFAT32Available() bool {
	_, err := exec.LookPath("mkfs.vfat")
	return err == nil
}

func MakeFAT32(device string) error {
	if out, err := exec.Command("mkfs.vfat", "-F", "32", device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create FAT32 filesystem: %w\noutput: %s", err, out)
	}
	return nil
}

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

func IsNTFSAvailable() bool {
	_, err := exec.LookPath("mkfs.ntfs")
	return err == nil
}

func MakeNTFS(device string) error {
	if out, err := exec.Command("mkfs.ntfs", "-Q", "-v", device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create NTFS filesystem: %w\noutput: %s", err, out)
	}
	return nil
}
