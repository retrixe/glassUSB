package main

import (
	"bytes"
	"fmt"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

// gptMetadataLBAs is the primary/backup GPT header (LBA 1) plus the default 128-entry
// partition array (32 sectors at 512 bytes/sector).
const gptMetadataLBAs = 33

// wipeStaleGPTMetadata removes leftover GPT headers and partition entry arrays so
// go-diskfs reads the new MBR table. go-diskfs prefers GPT over MBR when opening
// a disk; mbr.Table.Write only updates the four entries at the end of LBA 0.
func wipeStaleGPTMetadata(d *disk.Disk) error {
	rw, err := d.Backend.Writable()
	if err != nil {
		return err
	}
	lss := d.LogicalBlocksize
	if lss <= 0 {
		return fmt.Errorf("invalid logical block size: %d", lss)
	}
	diskLBAs := d.Size / lss
	if diskLBAs < gptMetadataLBAs*2+1 {
		return fmt.Errorf("disk is too small to wipe GPT metadata: %d logical blocks", diskLBAs)
	}

	zero := make([]byte, lss)
	for lba := int64(1); lba <= gptMetadataLBAs; lba++ {
		if _, err := rw.WriteAt(zero, lba*lss); err != nil {
			return fmt.Errorf("failed to wipe primary GPT metadata at LBA %d: %w", lba, err)
		}
	}
	for i := int64(0); i < gptMetadataLBAs; i++ {
		lba := diskLBAs - 1 - i
		if _, err := rw.WriteAt(zero, lba*lss); err != nil {
			return fmt.Errorf("failed to wipe backup GPT metadata at LBA %d: %w", lba, err)
		}
	}
	return nil
}

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
				// Apparently, Microsoft hates if you have an ESP on your GPT-bootable drive and another ESP on the main system...
				// Source: https://github.com/pbatard/rufus/blob/6d8fbf98305ff37eb531c45cbd6ff44563c53917/src/drive.c#L2479
				// Not verified myself, but preventatively align with Rufus by setting type MicrosoftBasicData.
				// Also, same reasoning for MBR applies here.
				{Start: uint64(primaryPartitionStart), End: uint64(primaryPartitionEnd), Type: gpt.MicrosoftBasicData, Name: "Windows ISO"},
			},
		}
	} else {
		if err := wipeStaleGPTMetadata(disk); err != nil {
			return fmt.Errorf("failed to wipe stale GPT metadata: %w", err)
		}
		table = &mbr.Table{
			Partitions: []*mbr.Partition{
				// Align with Rufus and don't use EFISystem, apparently Windows refuses to assign ESPs a drive letter either
				// https://github.com/pbatard/rufus/blob/6d8fbf98305ff37eb531c45cbd6ff44563c53917/src/drive.c#L2458
				{Start: uint32(primaryPartitionStart), Size: uint32(primaryPartitionSize), Type: mbr.Fat32LBA, Bootable: true},
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
				// Apparently, Microsoft hates if you have an ESP on your GPT-bootable drive and another ESP on the main system...
				// Source: https://github.com/pbatard/rufus/blob/6d8fbf98305ff37eb531c45cbd6ff44563c53917/src/drive.c#L2479
				// Not verified myself, but preventatively align with Rufus by setting type MicrosoftBasicData and attribute GPT_BASIC_DATA_ATTRIBUTE_NO_DRIVE_LETTER.
				{Start: uint64(secondaryPartitionStart), End: uint64(secondaryPartitionEnd), Type: gpt.MicrosoftBasicData, Name: "UEFI:NTFS", Attributes: 0x8000000000000000},
			},
		}
	} else {
		if err := wipeStaleGPTMetadata(disk); err != nil {
			return fmt.Errorf("failed to wipe stale GPT metadata: %w", err)
		}
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

	if _, err := disk.WritePartitionContents(partition, bytes.NewReader(UEFI_NTFS_IMG)); err != nil {
		return fmt.Errorf("failed to write UEFI:NTFS contents to partition: %w", err)
	}
	return nil
}
