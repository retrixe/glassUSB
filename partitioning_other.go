//go:build !linux

package main

func GetBlockDevicePartition(blockDevice string, partNumber int) string {
	panic("GetBlockDevicePartition is only implemented on Linux")
}
