package main

import (
	"bytes"
	"fmt"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

// FormatDiskForSinglePartition formats a disk with a single ESP partition spanning the entire disk.
func FormatDiskForSinglePartition(name string, useGpt bool) error {
	disk, err := diskfs.Open(name, diskfs.WithOpenMode(diskfs.ReadWrite))
	if err != nil {
		return fmt.Errorf("failed to open destination: %v", err)
	}
	defer disk.Close()

	primaryPartitionStart := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
	primaryPartitionSize := (disk.Size / disk.LogicalBlocksize) - primaryPartitionStart

	var table partition.Table
	if useGpt {
		// Reserve 2048 sectors at the end just like fdisk
		primaryPartitionSize -= 2048
		primaryPartitionEnd := primaryPartitionStart + primaryPartitionSize - 1
		table = &gpt.Table{
			ProtectiveMBR: true,
			Partitions: []*gpt.Partition{
				{Start: uint64(primaryPartitionStart), End: uint64(primaryPartitionEnd), Type: gpt.EFISystemPartition, Name: "EFI System"},
			},
		}
	} else {
		table = &mbr.Table{
			Partitions: []*mbr.Partition{
				{Start: uint32(primaryPartitionStart), Size: uint32(primaryPartitionSize), Type: mbr.EFISystem, Bootable: true},
			},
		}
	}

	if err := disk.Partition(table); err != nil {
		return fmt.Errorf("failed to create partition table: %w", err)
	}
	return nil
}

// FormatDiskForUEFINTFS formats a disk with 2 partitions:
// - A 1 MiB FAT32 ESP partition holding UEFI:NTFS
// - The remaining disk is spanned by an mbr.NTFS / gpt.MicrosoftBasicData partition
func FormatDiskForUEFINTFS(name string, useGpt bool) error {
	disk, err := diskfs.Open(name, diskfs.WithOpenMode(diskfs.ReadWrite))
	if err != nil {
		return fmt.Errorf("failed to open destination: %v", err)
	}
	defer disk.Close()

	// Windows partition
	primaryPartitionStart := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
	primaryPartitionSize := (disk.Size / disk.LogicalBlocksize) - (primaryPartitionStart * 2) // primaryPartitionStart + UEFI:NTFS partition
	primaryPartitionEnd := primaryPartitionStart + primaryPartitionSize - 1
	// UEFI:NTFS partition
	secondaryPartitionStart := primaryPartitionEnd + 1
	secondaryPartitionSize := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
	secondaryPartitionEnd := secondaryPartitionStart + secondaryPartitionSize - 1

	var table partition.Table
	if useGpt {
		// Reserve 2048 sectors at the end just like fdisk
		secondaryPartitionStart -= 2048
		secondaryPartitionEnd -= 2048
		primaryPartitionSize -= 2048
		primaryPartitionEnd := primaryPartitionStart + primaryPartitionSize - 1
		table = &gpt.Table{
			ProtectiveMBR: true,
			Partitions: []*gpt.Partition{
				{Start: uint64(primaryPartitionStart), End: uint64(primaryPartitionEnd), Type: gpt.MicrosoftBasicData, Name: "Windows ISO"},
				{Start: uint64(secondaryPartitionStart), End: uint64(secondaryPartitionEnd), Type: gpt.EFISystemPartition, Name: "EFI System"},
			},
		}
	} else {
		table = &mbr.Table{
			Partitions: []*mbr.Partition{
				{Start: uint32(primaryPartitionStart), Size: uint32(primaryPartitionSize), Type: mbr.NTFS, Bootable: true},
				{Start: uint32(secondaryPartitionStart), Size: uint32(secondaryPartitionSize), Type: mbr.EFISystem, Bootable: false},
			},
		}
	}

	if err := disk.Partition(table); err != nil {
		return fmt.Errorf("failed to create partition table: %w", err)
	}
	return nil
}

// WriteUEFINTFSToPartition writes the UEFI:NTFS image to the specified partition on the device.
func WriteUEFINTFSToPartition(name string, partition int) error {
	disk, err := diskfs.Open(name, diskfs.WithOpenMode(diskfs.ReadWrite))
	if err != nil {
		return fmt.Errorf("failed to open destination: %v", err)
	}
	defer disk.Close()

	if _, err := disk.WritePartitionContents(2, bytes.NewReader(UEFI_NTFS_IMG)); err != nil {
		return fmt.Errorf("failed to write UEFI:NTFS contents to partition: %w", err)
	}
	return nil
}
