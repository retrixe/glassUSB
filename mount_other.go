//go:build !linux

package main

import (
	"errors"
)

func MountPartition(partition string, mountPoint string) error {
	return errors.ErrUnsupported
}

func UnmountPartition(mountPoint string) error {
	return errors.ErrUnsupported
}
