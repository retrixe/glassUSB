package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Xmister/udf"
	"github.com/diskfs/go-diskfs"
	"github.com/retrixe/imprint/imaging"
)

var ErrInvalidWindowsISO = errors.New("this file is not recognised as a valid Windows ISO image in UDF format")

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

func GetISOContentSize(iso *udf.Udf) int64 {
	size := int64(0)
	for _, file := range iso.ReadDir(nil) {
		if file.IsDir() {
			size += getISOFileFolderSize(file)
		} else {
			size += file.Size()
		}
	}
	return size
}

func getISOFileFolderSize(folder udf.File) int64 {
	var size int64 = 0
	for _, f := range folder.ReadDir() {
		if f.IsDir() {
			size += getISOFileFolderSize(f)
		} else {
			size += f.Size()
		}
	}
	return size
}

func logProgressPerSecond(action string, progress *atomic.Int64, terminateProgress <-chan struct{}) {
	startTime := time.Now().UnixMilli()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		// FIXME: Accept a log function because we got the zenity wizard too...
		case <-ticker.C:
			print(imaging.FormatProgress(int(progress.Load()), time.Now().UnixMilli()-startTime, action, false) + "\r")
		case <-terminateProgress:
			println(imaging.FormatProgress(int(progress.Load()), time.Now().UnixMilli()-startTime, action, true))
			return
		}
	}
}

func ExtractISOToLocation(iso *udf.Udf, location string) error {
	progress := &atomic.Int64{}
	terminateProgress := make(chan struct{})
	go logProgressPerSecond("extracted", progress, terminateProgress)
	for _, file := range iso.ReadDir(nil) {
		if err := extractISOFileToLocation(file, location, progress); err != nil {
			return err
		}
	}
	terminateProgress <- struct{}{}
	return nil
}

func extractISOFileToLocation(file udf.File, location string, progress *atomic.Int64) error {
	if file.Name() == "install.wim" {
		return nil // FIXME: Skip install.wim
	}
	if file.IsDir() {
		folderPath := filepath.Join(location, file.Name())
		if err := os.MkdirAll(folderPath, file.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", folderPath, err)
		}
		for _, child := range file.ReadDir() {
			if err := extractISOFileToLocation(child, folderPath, progress); err != nil {
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
		n, err := io.CopyBuffer(newFile, file.NewReader(), buf)
		progress.Add(n) // FIXME: Break down CopyBuffer into a loop we control so we can update progress more smoothly
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

func ValidateISOAgainstLocation(iso *udf.Udf, location string) error {
	progress := &atomic.Int64{}
	terminateProgress := make(chan struct{})
	go logProgressPerSecond("validated", progress, terminateProgress)
	for _, file := range iso.ReadDir(nil) {
		if err := validateISOFileAgainstLocation(file, location, progress); err != nil {
			return err
		}
	}
	terminateProgress <- struct{}{}
	return nil
}

func validateISOFileAgainstLocation(file udf.File, location string, progress *atomic.Int64) error {
	if file.Name() == "install.wim" {
		return nil // FIXME: Skip install.wim
	}
	if file.IsDir() {
		// TODO: Check if there's extra files in location that are not in ISO
		folderPath := filepath.Join(location, file.Name())
		for _, child := range file.ReadDir() {
			if err := validateISOFileAgainstLocation(child, folderPath, progress); err != nil {
				return err
			}
		}
	} else {
		srcReader := file.NewReader()
		destFile, err := os.Open(filepath.Join(location, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", file.Name(), err)
		}
		defer destFile.Close()
		buf1 := make([]byte, 4*1024*1024)
		buf2 := make([]byte, 4*1024*1024)
		for {
			n1, err1 := srcReader.Read(buf1)
			if err1 != nil && err1 != io.EOF {
				return fmt.Errorf("failed to read file %s from ISO: %w", file.Name(), err1)
			}
			n2, err2 := io.ReadFull(destFile, buf2[:n1])
			if err2 != nil { // EOF should not happen here
				return fmt.Errorf("failed to read file %s from destination: %w", file.Name(), err2)
			}
			if !bytes.Equal(buf1[:n1], buf2[:n2]) {
				return fmt.Errorf("contents of file %s do not match the ISO", file.Name())
			}
			progress.Add(int64(n1))
			if err1 == io.EOF {
				break
			}
		}
		n, err := destFile.Read(buf2)
		if n > 0 || err != io.EOF {
			return fmt.Errorf("file %s on disk is larger than expected", file.Name())
		}
	}
	return nil
}
