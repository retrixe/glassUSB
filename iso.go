package main

import (
	"errors"
	"os"

	"github.com/Xmister/udf"
	"github.com/diskfs/go-diskfs"
)

var ErrInvalidWindowsISO = errors.New("this file is not recognised as a valid Windows ISO image")

func OpenWindowsISO(file *os.File) (*udf.Udf, error) {
	if !IsValidWindowsISO(file) {
		return nil, ErrInvalidWindowsISO
	}
	iso, err := udf.NewUdfFromReader(file)
	if err != nil {
		return nil, err
	}
	return iso, nil
}

func IsValidWindowsISO(file *os.File) bool {
	return !isFileDiskImage(file.Name()) && isFileUDF(file)
}

func isFileDiskImage(file string) bool {
	disk, err := diskfs.Open(file, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return false
	}
	defer disk.Close()
	table, err := disk.GetPartitionTable()
	return err == nil && table != nil
}

func isFileUDF(file *os.File) bool {
	defer func() { recover() }()
	iso, err := udf.NewUdfFromReader(file)
	return err == nil && iso != nil && len(iso.ReadDir(nil)) > 0
}
