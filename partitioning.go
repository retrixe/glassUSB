package main

import (
	"fmt"

	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

// FormatDiskWithUEFINTFS formats a disk with 2 partitions:
// - A 1 MiB FAT32 ESP partition holding UEFI:NTFS
// - The remaining disk is spanned by an mbr.NTFS / gpt.MicrosoftBasicData partition
func FormatDiskForUEFINTFS(disk *disk.Disk, useGpt bool) error {
	// exFAT/NTFS partition for Windows files
	primaryPartitionStart := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
	primaryPartitionSize := (disk.Size / disk.LogicalBlocksize) - (primaryPartitionStart * 2)
	primaryPartitionEnd := primaryPartitionStart + primaryPartitionSize - 1
	// UEFI:NTFS partition
	secondaryPartitionStart := primaryPartitionEnd + 1
	secondaryPartitionSize := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
	secondaryPartitionEnd := secondaryPartitionStart + secondaryPartitionSize - 1

	var table partition.Table
	// TODO: GPT doesn't seem to work properly for some reason
	if useGpt {
		// Reserve 2048 sectors at the end just like fdisk
		secondaryPartitionStart -= 2048
		secondaryPartitionEnd -= 2048
		primaryPartitionSize -= 2048
		primaryPartitionEnd := primaryPartitionStart + primaryPartitionSize - 1
		table = &gpt.Table{
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
