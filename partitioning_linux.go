//go:build linux

package main

import (
	"os"
	"strconv"
	"unsafe"

	"golang.org/x/sys/unix"
)

func GetBlockDevicePartition(blockDevice string, partNumber int) string {
	blockDevicePartition := blockDevice
	if blockDevice[len(blockDevice)-1] >= '0' && blockDevice[len(blockDevice)-1] <= '9' {
		blockDevicePartition += "p"
	}
	return blockDevicePartition + strconv.Itoa(partNumber)
}

func GetBlockDeviceSize(blockDevice string) (int64, error) {
	file, err := os.Open(blockDevice)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	var value uint64
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, file.Fd(), unix.BLKGETSIZE64, uintptr(unsafe.Pointer(&value)))
	if errno != 0 {
		return 0, errno
	}
	return int64(value), nil
}
