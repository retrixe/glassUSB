package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	_ "embed"

	"github.com/Xmister/udf"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

const version = "1.0.0-dev"

var vFlag = flag.Bool("v", false, "")
var versionFlag = flag.Bool("version", false, "Show version")

var flashFlagSet = flag.NewFlagSet("flash", flag.ExitOnError)
var gptFlag = flashFlagSet.Bool("gpt", false,
	"Use GPT partitioning\n"+
		"Note: Only compatible with UEFI systems i.e. PCs with Windows 8 or newer")
var primaryFsFlag = flashFlagSet.String("primary-fs", "exfat",
	"Filesystem to use. If not using FAT32, UEFI:NTFS will be installed, and all\n"+
		"ISO files will be stored on a single partition.\n"+
		"Available options: ")

/*
	 TODO: Support FAT32 + secondary-fs
		var secondaryFsFlag = flashFlagSet.String("secondary-fs", "exfat",
			"Filesystem to use for second partition if primary-fs=fat32 and ISO > 4GB\n"+
				"Options: exfat, ntfs")
		var disableValidationFlag = flashFlagSet.Bool("disable-validation", false,
			"Disable validation of written files")
*/

//go:embed binaries/uefi-ntfs.img
var UEFI_NTFS_IMG []byte

func init() {
	flag.Usage = func() {
		println("Usage: glassUSB [command] [options]")
		println("\nAvailable commands:")
		println("  flash       Flash a Windows ISO to a specific USB device.")
		println("\nOptions:")
		flag.PrintDefaults()
	}
	flashFlagSet.Usage = func() {
		println("Usage: glassUSB flash [options] <disk image file> <device path>")
		println("\nOptions:")
		flashFlagSet.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if (versionFlag != nil && *versionFlag) || (vFlag != nil && *vFlag) {
		println("glassUSB version v" + version)
		return
	} else if len(os.Args) >= 2 && os.Args[1] == "flash" {
		log.SetFlags(0)
		log.SetOutput(os.Stderr)
		log.SetPrefix("[glassUSB] ")

		// Look for prerequisites on system
		primaryFsFlagStruct := flashFlagSet.Lookup("primary-fs")
		_, exfatErr := exec.LookPath("mkfs.exfat")
		_, ntfsErr := exec.LookPath("mkfs.ntfs")
		if ntfsErr != nil && exfatErr != nil {
			log.Fatalln("Neither NTFS nor exFAT support were found on this system, exiting...")
		} else if ntfsErr != nil {
			primaryFsFlagStruct.Usage = primaryFsFlagStruct.Usage + "exfat"
		} else if exfatErr != nil {
			primaryFsFlagStruct.DefValue = "ntfs"
			primaryFsFlagStruct.Value.Set("ntfs")
			primaryFsFlagStruct.Usage = primaryFsFlagStruct.Usage + "ntfs"
		} else {
			primaryFsFlagStruct.Usage = primaryFsFlagStruct.Usage + "exfat, ntfs"
		}
		flashFlagSet.Parse(os.Args[2:])
		args := flashFlagSet.Args()
		if len(args) != 2 {
			flashFlagSet.Usage()
			os.Exit(1)
		}

		// Step 1: Read ISO
		file, err := os.Open(args[0])
		if err != nil {
			log.Fatalf("Failed to open ISO: %v", err)
		}
		defer file.Close()
		if !IsValidWindowsISO(file) {
			log.Fatalf("This file is not recognised as a valid Windows ISO image!")
		}
		iso, err := udf.NewUdfFromReader(file)
		if err != nil {
			log.Fatalf("Failed to read UDF filesystem on ISO: %v", err)
		}
		// TODO: Remove this
		for _, f := range iso.ReadDir(nil) {
			fmt.Printf("%s %-10d %-20s %v\n", f.Mode().String(), f.Size(), f.Name(), f.ModTime())
		}

		// Step 2: Check sources/install.wim if it exceeds 4 GB in size
		/* largeInstallWim := false
		for _, f := range iso.ReadDir(nil) {
			if f.Name() == "sources/install.wim" && f.Size() > 4*1024*1024*1024 {
				largeInstallWim = true
			}
		} */

		// Step 3: Open the block device and create a new partition table
		destStat, err := os.Stat(args[1])
		if err != nil {
			log.Fatalf("Failed to get info about destination: %v", err)
		}
		disk, err := diskfs.Open(args[1], diskfs.WithOpenMode(diskfs.ReadWrite))
		if err != nil {
			log.Fatalf("Failed to open destination: %v", err)
		}
		defer disk.Close()
		var table partition.Table
		// UEFI:NTFS partition
		primaryPartitionStart := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
		primaryPartitionSize := int64(1024*1024 /* 1 MiB */) / disk.LogicalBlocksize
		primaryPartitionEnd := primaryPartitionStart + primaryPartitionSize - 1
		// Windows partition
		secondaryPartitionStart := primaryPartitionEnd + 1
		secondaryPartitionEnd := (disk.Size / disk.LogicalBlocksize) - 1
		secondaryPartitionSize := secondaryPartitionEnd - secondaryPartitionStart + 1
		if gptFlag != nil && *gptFlag {
			// TODO: This doesn't apply reliably for some reason :/
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
			log.Fatalf("Failed to create partition table: %v", err)
		}

		// Step 4: Write UEFI:NTFS to first partition
		_, err = disk.WritePartitionContents(1, bytes.NewReader(UEFI_NTFS_IMG))
		if err != nil {
			log.Fatalf("Failed to write UEFI:NTFS to first partition: %v", err)
		}

		// Step 5: Mount second partition and create exFAT/NTFS partition depending on primaryFs
		//blockDevice := args[1]
		if destStat.Mode().Perm()&os.ModeDevice != 0 {
			// TODO: Support flashing to a loopback device
		} else {
			// TODO: Support flashing to a real device
		}

		return
	} else {
		flag.Usage()
		os.Exit(1)
	}
}
