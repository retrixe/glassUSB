package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

func ExtractISOToLocation(iso *udf.Udf, location string) error {
	for _, file := range iso.ReadDir(nil) {
		if err := extractISOFileToLocation(file, location); err != nil {
			return err
		}
	}
	return nil
}

func extractISOFileToLocation(file udf.File, location string) error {
	if file.Name() == "install.wim" {
		return nil // FIXME: Skip install.wim
	}
	if file.IsDir() {
		folderPath := filepath.Join(location, file.Name())
		if err := os.MkdirAll(folderPath, file.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", folderPath, err)
		}
		for _, child := range file.ReadDir() {
			if err := extractISOFileToLocation(child, folderPath); err != nil {
				return err
			}
		}
	} else {
		newFile, err := os.Create(filepath.Join(location, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", file.Name(), err)
		}
		defer newFile.Close()
		buf := make([]byte, 4*1024*1024)
		_, err = io.CopyBuffer(newFile, file.NewReader(), buf)
		if err != nil {
			return fmt.Errorf("failed to copy file %s: %w", file.Name(), err)
		}
		err = newFile.Sync()
		if err != nil {
			return fmt.Errorf("failed to sync file %s: %w", file.Name(), err)
		}
	}
	return nil
}
