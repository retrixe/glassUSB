//go:build darwin

package main

import (
	"fmt"
	"os/exec"
)

func IsFAT32Available() bool {
	_, err := exec.LookPath("newfs_msdos")
	return err == nil
}

func MakeFAT32(device string, label string) error {
	if out, err := exec.Command("newfs_msdos", "-F", "32", "-v", truncateString(label, 11), device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create FAT32 filesystem: %w\noutput: %s", err, out)
	}
	return nil
}

func IsExFATAvailable() bool {
	_, err := exec.LookPath("newfs_exfat")
	return err == nil
}

func MakeExFAT(device string, label string) error {
	if out, err := exec.Command("newfs_exfat", "-v", truncateString(label, 11), device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create exFAT filesystem: %w\noutput: %s", err, out)
	}
	return nil
}

func IsNTFSAvailable() bool {
	_, err := exec.LookPath("newfs_ntfs") // Provided by "Paragon NTFS for Mac OS X"
	return err == nil
}

func MakeNTFS(device string, label string) error {
	if out, err := exec.Command("newfs_ntfs", "-q", "-v", truncateString(label, 32), device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create NTFS filesystem: %w\noutput: %s", err, out)
	}
	return nil
}
