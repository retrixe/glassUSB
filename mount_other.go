//go:build !linux

package main

import (
	"errors"
)

func LoopMountFile(name string) (string, error) {
	return "", errors.ErrUnsupported
}

func LoopUnmountFile(loopDevice string) error {
	return errors.ErrUnsupported
}

func MountPartition(partition string, mountPoint string) error {
	return errors.ErrUnsupported
}

func UnmountPartition(mountPoint string) error {
	return errors.ErrUnsupported
}
