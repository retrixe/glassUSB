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
	"Use GPT partitioning instead of MBR.\n"+
		"Note: Only compatible with UEFI systems i.e. PCs with Windows 8 or newer")
var fsFlag = flashFlagSet.String("fs", "",
	"Filesystem to use for storing the USB flash drive contents.\n"+
		"\nIf using exFAT or NTFS, UEFI:NTFS will be installed to the EFI system partition,\n"+
		"and all ISO files will be placed on the exFAT/NTFS partition.\n"+
		"\nIf using FAT32, all ISO files will be placed on the EFI system partition. If\n"+
		"'sources/install.wim' is larger than 4 GB, the flash procedure will fail.\n"+
		//"\nIf using FAT32, all ISO files will be placed on the EFI system partition, If\n"+
		//"'sources/install.wim' is larger than 4 GB, a second exFAT/NTFS partition will be\n"+
		//"created to store the WIM file on.\n"+
		"\nAvailable options: ")

/*
	 TODO: Support FAT32 + secondary-fs
		var secondaryFsFlag = flashFlagSet.String("secondary-fs", "exfat",
			"Filesystem to use for second partition if primary-fs=fat32 and ISO > 4GB\n"+
				"Options: exfat, ntfs")
	 TODO: Support validation of written files
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

		// Look for prerequisites on system and change defaults accordingly
		fsFlagStruct := flashFlagSet.Lookup("fs")
		_, exfatErr := exec.LookPath("mkfs.exfat")
		_, ntfsErr := exec.LookPath("mkfs.ntfs")
		if ntfsErr == nil && exfatErr == nil {
			fsFlagStruct.Usage = fsFlagStruct.Usage + "exfat, ntfs"
			fsFlagStruct.DefValue = "exfat"
			fsFlagStruct.Value.Set("exfat")
		} else if ntfsErr != nil && exfatErr == nil {
			fsFlagStruct.Usage = fsFlagStruct.Usage + "exfat"
			fsFlagStruct.DefValue = "exfat"
			fsFlagStruct.Value.Set("exfat")
		} else if exfatErr != nil && ntfsErr == nil {
			fsFlagStruct.Usage = fsFlagStruct.Usage + "ntfs"
			fsFlagStruct.DefValue = "ntfs"
			fsFlagStruct.Value.Set("ntfs")
		} else {
			// TODO: FAT32 fallback
		}

		// Parse flags
		flashFlagSet.Parse(os.Args[2:])
		args := flashFlagSet.Args()
		if len(args) != 2 {
			flashFlagSet.Usage()
			os.Exit(1)
		} else if fsFlag == nil || (*fsFlag != "exfat" && *fsFlag != "ntfs" && *fsFlag != "") {
			log.Println("Invalid value provided for `-fs` flag!")
			flashFlagSet.Usage()
			os.Exit(1)
		} else if *fsFlag == "" {
			log.Fatalln("Neither NTFS nor exFAT support were found on this system, exiting...")
		} else if *fsFlag == "exfat" && exfatErr != nil {
			log.Fatalln("exFAT specified, but support is missing on this system, exiting...")
		} else if *fsFlag == "ntfs" && ntfsErr != nil {
			log.Fatalln("NTFS specified, but support is missing on this system, exiting...")
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
		// FIXME: Remove this
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
		// exFAT/NTFS partition for Windows files
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

		// Step 5: Create exFAT/NTFS partition depending on fs flag
		blockDevice := args[1]
		// Mount regular files as loopback devices (TODO: Guard this behind a flag?)
		if destStat.Mode().IsRegular() {
			loopDevice, err := exec.Command("losetup", "-f").CombinedOutput()
			if err != nil {
				log.Fatalf("Failed to find free loop device: %v\nOutput: %s", err, loopDevice)
			}
			blockDevice = string(bytes.TrimSpace(loopDevice))
			out, err := exec.Command("losetup", "-P", blockDevice, args[1]).CombinedOutput()
			if err != nil {
				log.Fatalf("Failed to set up loop device: %v\nOutput: %s", err, out)
			}
			defer func() {
				if out, err := exec.Command("losetup", "-d", blockDevice).CombinedOutput(); err != nil {
					log.Printf("Failed to detach loop device: %v\nOutput: %s", err, out)
				}
			}()
		}
		blockDevicePartition := blockDevice
		if blockDevice[len(blockDevice)-1] >= '0' && blockDevice[len(blockDevice)-1] <= '9' {
			blockDevicePartition += "p2"
		} else {
			blockDevicePartition += "2"
		}
		if out, err := exec.Command("mkfs."+(*fsFlag), blockDevicePartition).CombinedOutput(); err != nil {
			log.Fatalf("Failed to create filesystem: %v\nOutput: %s", err, out)
		}

		// Step 6: Mount exFAT/NTFS partition, defer unmount
		mountPoint, err := os.MkdirTemp(os.TempDir(), "glassusb-")
		if err != nil {
			log.Fatalf("Failed to create mount point: %v", err)
		}
		defer os.Remove(mountPoint)
		if out, err := exec.Command("mount", blockDevicePartition, mountPoint).CombinedOutput(); err != nil {
			log.Fatalf("Failed to mount partition: %v\nOutput: %s", err, out)
		}
		defer func() {
			if out, err := exec.Command("umount", mountPoint).CombinedOutput(); err != nil {
				log.Printf("Failed to unmount partition: %v\nOutput: %s", err, out)
			}
		}()

		// FIXME: Step 7: Unpack Windows ISO contents to exFAT/NTFS partition

		// FIXME: Step 8: Write MBR to device for exFAT/NTFS boot using `ms-sys`

		return
	} else {
		flag.Usage()
		os.Exit(1)
	}
}
