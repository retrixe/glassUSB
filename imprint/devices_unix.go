//go:build !darwin && !windows

package imprint

import (
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

// Source: https://github.com/retrixe/imprint/blob/main/app/devices_unix.go

// UnmountDevice unmounts a block device's partitons before flashing to it.
func UnmountDevice(device string) error {
	// Check if device is mounted.
	stat, err := os.Stat(device)
	if err != nil {
		return err
	} else if stat.Mode().Type()&fs.ModeDevice == 0 {
		return ErrNotBlockDevice
	}
	// Discover mounted device partitions.
	// TODO: Replace with syscall?
	mounts, err := exec.Command("mount").Output()
	if err != nil {
		return err
	}
	// Unmount device partitions.
	for _, mount := range strings.Split(string(mounts), "\n") {
		if strings.HasPrefix(mount, device) {
			partition := strings.Fields(mount)[0]
			// TODO: Use syscall.Unmount instead?
			err = exec.Command("umount", partition).Run()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
