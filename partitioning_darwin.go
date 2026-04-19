//go:build darwin

package main

import (
	"os"
	"strconv"

	"github.com/diskfs/go-diskfs"
	"golang.org/x/sys/unix"
)

func GetBlockDevicePartition(blockDevice string, partNumber int) string {
	return blockDevice + "s" + strconv.Itoa(partNumber)
}

func GetBlockDeviceSize(blockDevice string) (int64, error) {
	file, err := os.Open(blockDevice)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	fd := file.Fd()

	blockSize, err := unix.IoctlGetInt(int(fd), diskfs.DKIOCGETBLOCKSIZE)
	if err != nil {
		return 0, err
	}

	blockCount, err := unix.IoctlGetInt(int(fd), diskfs.DKIOCGETBLOCKCOUNT)
	if err != nil {
		return 0, err
	}
	return int64(blockSize) * int64(blockCount), nil
}
