package main

import (
	"bytes"
	"fmt"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

// FormatDiskWithUEFINTFS formats a disk with 2 partitions:
// - A 1 MiB FAT32 ESP partition holding UEFI:NTFS
// - The remaining disk is spanned by an mbr.NTFS / gpt.MicrosoftBasicData partition
func FormatDiskWithUEFINTFS(name string, useGpt bool) error {
	disk, err := diskfs.Open(name, diskfs.WithOpenMode(diskfs.ReadWrite))
	if err != nil {
		return fmt.Errorf("failed to open destination: %w", err)
	}
	defer disk.Close()

	// UEFI:NTFS partition
	primaryPartitionStart := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
	primaryPartitionSize := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
	primaryPartitionEnd := primaryPartitionStart + primaryPartitionSize - 1
	// exFAT/NTFS partition for Windows files
	secondaryPartitionStart := primaryPartitionEnd + 1
	secondaryPartitionEnd := (disk.Size / disk.LogicalBlocksize) - 1
	secondaryPartitionSize := secondaryPartitionEnd - secondaryPartitionStart + 1

	var table partition.Table
	// TODO: GPT doesn't seem to work properly for some reason
	if useGpt {
		secondaryPartitionEnd = secondaryPartitionEnd - 2048 // Reserve these at the end like fdisk
		secondaryPartitionSize = secondaryPartitionEnd - secondaryPartitionStart + 1
		table = &gpt.Table{
			Partitions: []*gpt.Partition{
				{Start: uint64(primaryPartitionStart), End: uint64(primaryPartitionEnd), Type: gpt.EFISystemPartition, Name: "EFI System"},
				{Start: uint64(secondaryPartitionStart), End: uint64(secondaryPartitionEnd), Type: gpt.MicrosoftBasicData, Name: "Windows ISO"},
			},
		}
	} else {
		table = &mbr.Table{
			Partitions: []*mbr.Partition{
				{Start: uint32(primaryPartitionStart), Size: uint32(primaryPartitionSize), Type: mbr.EFISystem, Bootable: true},
				{Start: uint32(secondaryPartitionStart), Size: uint32(secondaryPartitionSize), Type: mbr.NTFS, Bootable: false},
			},
		}
	}

	err = disk.Partition(table)
	if err != nil {
		return fmt.Errorf("failed to create partition table: %w", err)
	}

	// Write UEFI:NTFS image to first partition
	_, err = disk.WritePartitionContents(1, bytes.NewReader(UEFI_NTFS_IMG))
	if err != nil {
		return fmt.Errorf("failed to write UEFI:NTFS to first partition: %w", err)
	}
	return nil
}
