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

func MakeFAT32(device string, label string) error {
	if out, err := exec.Command("mkfs.vfat", "-F", "32", "-v", "-n", label, device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create FAT32 filesystem: %w\noutput: %s", err, out)
	}
	return nil
}

func IsExFATAvailable() bool {
	_, err := exec.LookPath("mkfs.exfat")
	return err == nil
}

func MakeExFAT(device string, label string) error {
	if out, err := exec.Command("mkfs.exfat", "-v", "-L", label, device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create exFAT filesystem: %w\noutput: %s", err, out)
	}
	return nil
}

func IsNTFSAvailable() bool {
	_, err := exec.LookPath("mkfs.ntfs")
	return err == nil
}

func MakeNTFS(device string, label string) error {
	// TODO: These are ntfs-3g specific parameters, update when ntfsplus is a thing
	if out, err := exec.Command("mkfs.ntfs", "-Q", "-v", "-L", label, device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create NTFS filesystem: %w\noutput: %s", err, out)
	}
	return nil
}
