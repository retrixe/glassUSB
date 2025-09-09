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
		supportedFilesystems := []string{}
		if IsExFATAvailable() {
			supportedFilesystems = append(supportedFilesystems, "exfat")
		}
		if IsNTFSAvailable() {
			supportedFilesystems = append(supportedFilesystems, "ntfs")
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

		// Step 1: Read ISO
		file, err := os.Open(args[0])
		if err != nil {
			log.Fatalf("Failed to open ISO: %v", err)
		}
		defer file.Close()
		iso, err := OpenWindowsISO(file)
		if err != nil {
			log.Fatalf("Failed to read UDF filesystem on ISO: %v", err)
		}

		// Step 2: Check sources/install.wim if it exceeds 4 GB in size
		/* largeInstallWim := false
		for _, f := range iso.ReadDir(nil) {
			if f.Name() == "sources/install.wim" && f.Size() > 4*1024*1024*1024 {
				largeInstallWim = true
			}
		} */

		// Step 3: Open the block device and create a new partition table
		// Step 4: Write UEFI:NTFS to first partition
		destStat, err := os.Stat(args[1])
		if err != nil {
			log.Fatalf("Failed to get info about destination: %v", err)
		}
		err = FormatDiskWithUEFINTFS(args[1], gptFlag != nil && *gptFlag)
		if err != nil {
			log.Fatalf("Failed to format disk: %v", err)
		}

		// Step 5a: Mount a regular file destination as a loopback device
		// TODO: Guard this behind a flag?
		blockDevice := args[1]
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

		// Step 5b: Create exFAT/NTFS partition depending on fs flag
		windowsPartition := GetBlockDevicePartition(blockDevice, 2)
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

		// Step 6: Mount exFAT/NTFS partition, defer unmount
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

		// Step 7: Unpack Windows ISO contents to exFAT/NTFS partition
		if err := ExtractISOToLocation(iso, mountPoint); err != nil {
			log.Fatalf("Failed to extract ISO contents: %v", err)
		}

		// FIXME: Step 8: Write MBR to device for exFAT/NTFS boot using `ms-sys`

		return
	} else {
		flag.Usage()
		os.Exit(1)
	}
}
