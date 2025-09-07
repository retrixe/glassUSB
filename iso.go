package main

import (
	"os"

	"github.com/Xmister/udf"
	"github.com/diskfs/go-diskfs"
)

func IsFileDiskImage(file string) bool {
	disk, err := diskfs.Open(file, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return false
	}
	defer disk.Close()
	table, err := disk.GetPartitionTable()
	return err == nil && table != nil
}

func IsFileUDF(file *os.File) bool {
	defer func() { recover() }()
	iso, err := udf.NewUdfFromReader(file)
	return err == nil && iso != nil && len(iso.ReadDir(nil)) > 0
}

func IsValidWindowsISO(file *os.File) bool {
	return !IsFileDiskImage(file.Name()) && IsFileUDF(file)
}
