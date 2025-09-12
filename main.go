package main

import (
	"flag"
	"log"
	"os"
	"slices"
	"strings"

	_ "embed"
)

const version = "1.0.0-dev"

var vFlag = flag.Bool("v", false, "")
var versionFlag = flag.Bool("version", false, "Show version")

var flashFlagSet = flag.NewFlagSet("flash", flag.ExitOnError)
var gptFlag = flashFlagSet.Bool("gpt", false,
	"EXPERIMENTAL: Use GPT partitioning instead of MBR.\n"+
		"Note: Only compatible with UEFI systems i.e. PCs with Windows 8 or newer")
var fsFlag = flashFlagSet.String("fs", "",
	"Filesystem to use for storing the USB flash drive contents.\n"+
		"\nIf using NTFS or exFAT, UEFI:NTFS will be installed to the EFI system partition,\n"+
		"and all ISO files will be placed on the NTFS/exFAT partition.\n"+
		"Note: Drives formatted with exFAT will not boot on PCs with Secure Boot enabled.\n"+
		"\nIf using FAT32, all ISO files will be placed on the EFI system partition. If\n"+
		"'sources/install.wim' is larger than 4 GB, the flash procedure will fail.\n"+
		//"\nIf using FAT32, all ISO files will be placed on the EFI system partition, If\n"+
		//"'sources/install.wim' is larger than 4 GB, a second NTFS/exFAT partition will be\n"+
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
		supportedFilesystems := []string{}
		if IsNTFSAvailable() {
			supportedFilesystems = append(supportedFilesystems, "ntfs")
		} else {
			// FIXME: Remove once FAT32 support is added, replace with supportedFilesystems += "fat32" after NTFS
			log.Println("Warning: NTFS support not found on this system, falling back to exFAT.")
			log.Println("Warning: exFAT partitions will not boot on PCs with Secure Boot on.")
		}
		if IsExFATAvailable() {
			supportedFilesystems = append(supportedFilesystems, "exfat")
		}
		if len(supportedFilesystems) > 0 {
			fsFlagStruct.DefValue = supportedFilesystems[0]
			fsFlagStruct.Value.Set(supportedFilesystems[0])
			fsFlagStruct.Usage = fsFlagStruct.Usage + strings.Join(supportedFilesystems, ", ")
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
		} else if *fsFlag == "exfat" && !slices.Contains(supportedFilesystems, "exfat") {
			log.Fatalln("exFAT specified, but support is missing on this system, exiting...")
		} else if *fsFlag == "ntfs" && !slices.Contains(supportedFilesystems, "ntfs") {
			log.Fatalln("NTFS specified, but support is missing on this system, exiting...")
		}

		// Warn exFAT does not work with Secure Boot enabled.
		if *fsFlag == "exfat" {
			log.Println("Warning: Drives formatted with exFAT (--fs=exfat) will not boot on PCs with Secure Boot enabled.")
		}

		// Step 1: Read ISO
		log.Println("Phase 1/6: Reading ISO")
		file, err := os.Open(args[0])
		if err != nil {
			log.Fatalf("Failed to open ISO: %v", err)
		}
		defer file.Close()
		iso, err := OpenWindowsISO(file)
		if err != nil {
			log.Fatalf("Failed to read UDF filesystem on ISO: %v", err)
		}
		/* largeFiles := false
		for _, f := range iso.ReadDir(nil) {
			if f.Name() == "sources/install.wim" && f.Size() > 4*1024*1024*1024 {
				largeFiles = true
			}
		} */

		// Step 2: Open the block device and create a new partition table
		log.Println("Phase 2/6: Partitioning destination drive")
		destStat, err := os.Stat(args[1])
		if err != nil {
			log.Fatalf("Failed to get info about destination: %v", err)
		}
		err = FormatDiskForUEFINTFS(args[1], gptFlag != nil && *gptFlag)
		if err != nil {
			log.Fatalf("Failed to format disk: %v", err)
		}
		blockDevice := args[1]
		windowsPartition := GetBlockDevicePartition(blockDevice, 1)
		uefiNtfsPartition := GetBlockDevicePartition(blockDevice, 2)

		// Step 3: Write UEFI:NTFS to second partition
		log.Println("Phase 3/6: Writing UEFI:NTFS bootloader")
		err = WriteUEFINTFSToPartition(uefiNtfsPartition)
		if err != nil {
			log.Fatalf("Failed to write UEFI bootloader to second partition: %v", err)
		}

		// Step 4a: Mount a regular file destination as a loopback device
		// TODO: Guard this behind a flag?
		if destStat.Mode().IsRegular() {
			loopDevice, err := LoopMountFile(args[1])
			if err != nil {
				log.Fatalf("Failed to set up loop device: %v", err)
			}
			blockDevice = loopDevice
			defer func() {
				if err := LoopUnmountFile(blockDevice); err != nil {
					log.Printf("Failed to detach loop device: %v", err)
				}
			}()
		}

		// Step 4b: Create NTFS/exFAT partition depending on fs flag
		switch *fsFlag {
		case "exfat":
			if err := MakeExFAT(windowsPartition); err != nil {
				log.Fatalf("Failed to create exFAT filesystem: %v", err)
			}
		case "ntfs":
			if err := MakeNTFS(windowsPartition); err != nil {
				log.Fatalf("Failed to create NTFS filesystem: %v", err)
			}
		}

		// Step 5: Mount NTFS/exFAT partition, defer unmount
		log.Println("Phase 4/6: Mounting sources partition")
		func() {
			mountPoint, err := os.MkdirTemp(os.TempDir(), "glassusb-")
			if err != nil {
				log.Fatalf("Failed to create mount point: %v", err)
			}
			defer os.Remove(mountPoint)
			if err := MountPartition(windowsPartition, mountPoint); err != nil {
				log.Fatalf("Failed to mount partition: %v", err)
			}
			defer func() {
				if err := UnmountPartition(mountPoint); err != nil {
					log.Printf("Failed to unmount partition: %v", err)
				}
			}()

			// Step 6: Unpack Windows ISO contents to NTFS/exFAT partition
			log.Println("Phase 5/6: Extracting ISO to sources partition")
			if err := ExtractISOToLocation(iso, mountPoint); err != nil {
				log.Fatalf("Failed to extract ISO contents: %v", err)
			}
		}()

		// Step 7: Write MBR to device for NTFS/exFAT boot using `ms-sys`
		log.Println("Phase 6/6: Writing MBR bootloader")
		if err := WriteMBRToPartition(windowsPartition); err != nil {
			log.Fatalf("Failed to write MBR bootloader: %v", err)
		}
		if err := WriteMBRToPartition(blockDevice); err != nil {
			log.Fatalf("Failed to write MBR bootloader: %v", err)
		}

		return
	} else {
		flag.Usage()
		os.Exit(1)
	}
}
