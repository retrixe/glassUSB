package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	_ "embed"

	"github.com/ncruces/zenity"
	"github.com/retrixe/imprint/imaging"
)

const version = "1.0.0-dev"

var vFlag = flag.Bool("v", false, "")
var versionFlag = flag.Bool("version", false, "Show version")

func mainUsage() {
	println("Usage: glassUSB [command] [options]")
	println("\nAvailable commands:")
	println("  flash       Flash a Windows ISO to a specific USB device.")
	println("  wizard      Start the GUI wizard for flashing Windows ISOs to a USB device.")
	println("\nOptions:")
	flag.PrintDefaults()
}

var flashFlagSet = flag.NewFlagSet("flash", flag.ExitOnError)
var gptFlag = flashFlagSet.Bool("gpt", false,
	"EXPERIMENTAL: Use GPT partitioning instead of MBR.\n"+
		"Note: Only compatible with UEFI systems i.e. PCs with Windows 8 or newer")
var fsFlag = flashFlagSet.String("fs", "",
	"Filesystem to use for storing the USB flash drive contents.\n"+
		"\nIf using NTFS or exFAT, UEFI:NTFS will be installed to an EFI system partition,\n"+
		"and all ISO files will be placed on the NTFS/exFAT partition.\n"+
		"Note: Drives formatted with exFAT will not boot on PCs with Secure Boot enabled.\n"+
		"\nIf using FAT32, all ISO files will be placed on a FAT32 EFI system partition. If\n"+
		"'sources/install.wim' is larger than 4 GB, the flash procedure will fail.\n"+
		//"\nIf using FAT32, all ISO files will be placed on the EFI system partition, If\n"+
		//"'sources/install.wim' is larger than 4 GB, a second NTFS/exFAT partition will be\n"+
		//"created to store the WIM file on.\n"+
		"\nAvailable options: ")
var skipValidationFlag = flashFlagSet.Bool("skip-validation", false,
	"Skip validation of written files")

func flashUsage() {
	println("Usage: glassUSB flash [options] <disk image file> <device path>")
	println("\nFlash a Windows ISO to a specific USB device.")
	println("\nOptions:")
	flashFlagSet.PrintDefaults()
}

func flashWizardUsage() {
	println("Usage: glassUSB wizard [options]")
	println("\nStart the GUI wizard for flashing Windows ISOs to a USB device.")
	println("\nOptions:")
	flashFlagSet.PrintDefaults()
}

/*
	 TODO: Support FAT32 + secondary-fs or some kind of splitting
		var secondaryFsFlag = flashFlagSet.String("secondary-fs", "exfat",
			"Filesystem to use for second partition if primary-fs=fat32 and ISO > 4GB\n"+
				"Options: exfat, ntfs")
*/

//go:embed binaries/uefi-ntfs.img
var UEFI_NTFS_IMG []byte

func init() {
	flag.Usage = mainUsage
	flashFlagSet.Usage = flashUsage
}

func main() {
	flag.Parse()
	if (versionFlag != nil && *versionFlag) || (vFlag != nil && *vFlag) {
		println("glassUSB version v" + version)
		return
	} else if len(os.Args) >= 2 && os.Args[1] == "flash" {
		flashCommand(false)
	} else if len(os.Args) >= 2 && os.Args[1] == "wizard" {
		flashFlagSet.Usage = flashWizardUsage
		flashCommand(true)
	} else {
		flag.Usage()
		os.Exit(1)
	}
}

func flashCommand(wizard bool) {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	log.SetPrefix("[glassUSB] ")
	var dlg zenity.ProgressDialog
	logProgress := func(message string) {
		log.Println(message)
		if dlg != nil {
			dlg.Text(message)
		}
	}
	logWarn := func(format string, v ...any) {
		log.Printf(format, v...)
		if wizard {
			zenity.Warning(fmt.Sprintf(format, v...),
				zenity.Width(640),
				zenity.WindowIcon(zenity.WarningIcon),
				zenity.Title("glassUSB Media Creation Wizard"),
				zenity.Icon(zenity.WarningIcon),
				zenity.OKLabel("Continue"))
		}
	}
	logFatal := func(format string, v ...any) {
		if wizard {
			zenity.Error(fmt.Sprintf(format, v...),
				zenity.Width(640),
				zenity.WindowIcon(zenity.ErrorIcon),
				zenity.Title("glassUSB Media Creation Wizard"),
				zenity.Icon(zenity.ErrorIcon),
				zenity.OKLabel("Exit"))
		}
		log.Panicf(format, v...)
	}

	// Look for prerequisites on system and change defaults accordingly
	fsFlagStruct := flashFlagSet.Lookup("fs")
	supportedFilesystems := []string{}
	fullySupportedFsAvailable := false
	if IsNTFSAvailable() {
		supportedFilesystems = append(supportedFilesystems, "ntfs")
		fullySupportedFsAvailable = true
	}
	if IsExFATAvailable() {
		supportedFilesystems = append(supportedFilesystems, "exfat")
	}
	if IsFAT32Available() {
		supportedFilesystems = append(supportedFilesystems, "fat32")
	}
	if len(supportedFilesystems) > 0 {
		fsFlagStruct.DefValue = supportedFilesystems[0]
		fsFlagStruct.Value.Set(supportedFilesystems[0])
		fsFlagStruct.Usage = fsFlagStruct.Usage + strings.Join(supportedFilesystems, ", ")
	}

	// Parse flags
	flashFlagSet.Parse(os.Args[2:])
	args := flashFlagSet.Args()
	if (wizard && len(args) != 0) || (!wizard && len(args) != 2) {
		flashFlagSet.Usage()
		os.Exit(1)
	} else if fsFlag == nil || (*fsFlag != "exfat" && *fsFlag != "ntfs" && *fsFlag != "fat32" && *fsFlag != "") {
		logFatal("Invalid value provided for `-fs` flag!")
		flashFlagSet.Usage()
		os.Exit(1)
	} else if *fsFlag == "" {
		logFatal("This system does not have any filesystem drivers supported by glassUSB, exiting...")
	} else if *fsFlag == "exfat" && !slices.Contains(supportedFilesystems, "exfat") {
		logFatal("exFAT specified, but support is missing on this system, exiting...")
	} else if *fsFlag == "ntfs" && !slices.Contains(supportedFilesystems, "ntfs") {
		logFatal("NTFS specified, but support is missing on this system, exiting...")
	} else if *fsFlag == "fat32" && !slices.Contains(supportedFilesystems, "fat32") {
		logFatal("FAT32 specified, but support is missing on this system, exiting...")
	}

	// Warn about exFAT and FAT32 limitations
	addendum := "If you encounter any issues, try installing NTFS drivers on your system (if using Linux), and using NTFS instead."
	if fullySupportedFsAvailable {
		addendum = "If you encounter any issues, try using NTFS instead."
	}
	switch *fsFlag {
	case "exfat":
		logWarn("%s %s", "Warning: Drives formatted with exFAT (--fs=exfat) will not boot on PCs with Secure Boot enabled.", addendum)
	case "fat32":
		logWarn("%s %s", "Warning: Using FAT32 (--fs=fat32) may cause flashing to fail for ISOs larger than 4 GB in size.", addendum)
	}

	// If using the wizard, prompt user for ISO and device
	if wizard {
		err := zenity.Question(`This wizard will guide you through the process of creating a Windows installation USB drive.

Make sure you have a spare USB flash drive connected to your computer (>8 GB recommended for Windows 11), and a Windows installation ISO downloaded.

Press 'Continue' to select the Windows ISO you downloaded. Supported versions of Windows include Vista, 7 and newer.`,
			zenity.Width(640),
			zenity.Height(480),
			zenity.WindowIcon(zenity.InfoIcon),
			zenity.Title("glassUSB Media Creation Wizard"),
			zenity.Icon(zenity.InfoIcon),
			zenity.CancelLabel("Exit"),
			zenity.OKLabel("Continue"))
		if err != nil {
			log.Panicf("Failed to continue with wizard: %v", err)
		}

		wd, err := os.Getwd()
		if err != nil {
			logFatal("Failed to open file dialog: %v", err)
		}
		isoPath, err := zenity.SelectFile(
			zenity.WindowIcon(zenity.QuestionIcon),
			zenity.Title("glassUSB - Select Windows ISO"),
			zenity.Filename(wd+string(os.PathSeparator)),
			zenity.FileFilters{
				{Name: "ISO Images", Patterns: []string{"*.iso", "*.img"}},
				{Name: "All Files", Patterns: []string{"*"}},
			},
		)
		if err != nil {
			log.Panicf("Failed to continue with wizard: %v", err)
		}

		var device, deviceName string
		for {
			devices, err := imaging.GetDevices(imaging.SystemPlatform)
			if err != nil {
				logFatal("Failed to get connected drives: %v", err)
			} else if len(devices) == 0 {
				err = zenity.Error("Failed to find any USB devices connected to your computer.\n\n"+
					"Please connect a USB flash drive and try again.",
					zenity.Width(640),
					zenity.WindowIcon(zenity.ErrorIcon),
					zenity.Title("glassUSB - Select target USB drive"),
					zenity.Icon(zenity.ErrorIcon),
					zenity.OKLabel("Exit"),
					zenity.ExtraButton("Rescan devices"))
				if err == nil {
					log.Fatalln("No USB devices connected, exiting...")
				} else if !errors.Is(err, zenity.ErrExtraButton) {
					log.Panicf("Failed to continue with wizard: %v", err)
				}
				continue
			}

			stringifiedDevices := make([]string, len(devices))
			for index, device := range devices {
				if device.Model == "" {
					stringifiedDevices[index] = device.Name + " (" + device.Size + ")"
				} else {
					stringifiedDevices[index] = device.Name + " (" + device.Model + ", " + device.Size + ")"
				}
			}
			device, err = zenity.List("Select a target device to flash the Windows ISO to:\n\n"+
				"⚠️ Warning: All data on the USB drive you select will be ERASED!\n"+
				"If you have any files stored on the drive, back them up before proceeding!",
				stringifiedDevices,
				zenity.Width(640),
				zenity.Height(480),
				zenity.WindowIcon(zenity.QuestionIcon),
				zenity.Title("glassUSB - Select target USB drive"),
				zenity.DisallowEmpty(),
				zenity.RadioList(),
				zenity.OKLabel("Continue"),
				zenity.ExtraButton("Rescan devices"),
			)
			if errors.Is(err, zenity.ErrExtraButton) {
				continue
			} else if err != nil {
				log.Panicf("Failed to continue with wizard: %v", err)
			} else if device != "" {
				deviceName = device[:strings.LastIndex(device, " (")]
				break
			}
		}

		err = zenity.Question(`The following Windows ISO will be flashed to the target USB drive:

`+isoPath+`

The following device will be converted into a Windows installation USB drive:

`+device+`

⚠️ Warning: All data on this USB drive will be ERASED! If you have any files stored on the drive, cancel here and back them up before proceeding to flash!`,
			zenity.Width(640),
			zenity.Height(480),
			zenity.WindowIcon(zenity.InfoIcon),
			zenity.Title("glassUSB - Confirm Flash and Data Wipe"),
			zenity.Icon(zenity.InfoIcon),
			zenity.CancelLabel("Exit"),
			zenity.OKLabel("Continue"))
		if err != nil {
			log.Panicf("Failed to continue with wizard: %v", err)
		}

		dlg, err = zenity.Progress(
			zenity.Width(640),
			zenity.WindowIcon(zenity.InfoIcon),
			zenity.Title("glassUSB Media Creation Wizard"),
			zenity.Icon(zenity.NoIcon),
			zenity.Pulsate(),
			// TODO: Could we use TimeRemaining at the flash stage
			zenity.NoCancel(), // TODO: Once cancellation is implemented we ball
			zenity.CancelLabel("Cancel"),
			zenity.OKLabel("Finish"))
		if err != nil {
			log.Panicf("Failed to continue with wizard: %v", err)
		}
		defer dlg.Close()

		args = []string{isoPath, deviceName}
	}

	totalPhasesNum := 7
	if *gptFlag {
		totalPhasesNum-- // Skip MBR writing phase
	}
	if *skipValidationFlag {
		totalPhasesNum-- // Skip validation phase
	}
	if *fsFlag == "fat32" {
		totalPhasesNum-- // Skip UEFI:NTFS writing phase
	}
	totalPhases := strconv.Itoa(totalPhasesNum)
	currentPhase := 0

	// Step 1: Read ISO
	currentPhase++
	logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Reading ISO")
	file, err := os.Open(args[0])
	if err != nil {
		logFatal("Failed to open ISO: %v", err)
	}
	defer file.Close()
	srcStat, err := file.Stat()
	if err != nil {
		logFatal("Failed to stat ISO file: %v", err)
	}
	iso, err := OpenWindowsISO(file)
	if err != nil {
		logFatal("Failed to read UDF filesystem on ISO: %v", err)
	}
	//totalSize := GetISOContentSize(iso)
	//log.Println("Total ISO size:", strconv.Itoa(int(totalSize)), "bytes",
	//	"("+imaging.BytesToString(int(totalSize), false)+", "+imaging.BytesToString(int(totalSize), true)+")")

	// Step 2: Open the block device and create a new partition table
	blockDevice := args[1]
	currentPhase++
	logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Partitioning destination drive")
	destStat, err := os.Stat(blockDevice)
	if err != nil {
		logFatal("Failed to get info about destination: %v", err)
	} else if destStat.Mode().Type()&os.ModeDevice == 0 {
		allowRegularDest, exists := os.LookupEnv("__GLASSUSB_DEBUG_ALLOW_REGULAR_DEST")
		if !exists || (allowRegularDest != "true" && allowRegularDest != "1") {
			logFatal("Destination %s is not a valid block device!", blockDevice)
		}
	}
	blockDeviceSize, err := GetBlockDeviceSize(blockDevice)
	if err != nil {
		logFatal("Failed to get size of destination: %v", err)
	} else if srcStat.Size() > blockDeviceSize {
		disableSizeCheck, exists := os.LookupEnv("__GLASSUSB_DEBUG_DISABLE_SIZE_CHECK")
		if !exists || (disableSizeCheck != "true" && disableSizeCheck != "1") {
			logFatal("Cannot write ISO to destination: ISO size (%s) is larger than device size (%s)!",
				imaging.BytesToString(int(srcStat.Size()), true),
				imaging.BytesToString(int(blockDeviceSize), true))
		}
	}
	err = imaging.UnmountDevice(blockDevice)
	if err != nil && err != imaging.ErrNotBlockDevice { // Ignore non-block-device error here
		logFatal("Failed to unmount destination device: %v", err)
	}
	if *fsFlag == "fat32" {
		err = FormatDiskForSinglePartition(blockDevice, gptFlag != nil && *gptFlag)
	} else {
		err = FormatDiskForUEFINTFS(blockDevice, gptFlag != nil && *gptFlag)
	}
	if err != nil {
		logFatal("Failed to format disk: %v", err)
	}

	// Step 3: Write UEFI:NTFS to second partition
	if *fsFlag != "fat32" {
		currentPhase++
		logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Writing UEFI:NTFS bootloader")
		err = WriteUEFINTFSToPartition(blockDevice, 2)
		if err != nil {
			logFatal("Failed to write UEFI bootloader to second partition: %v", err)
		}
	}

	// Step 4: Format primary partition depending on fs flag
	currentPhase++
	logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Creating sources partition")
	primaryPartition := GetBlockDevicePartition(blockDevice, 1)
	switch *fsFlag {
	case "exfat":
		if err := MakeExFAT(primaryPartition); err != nil {
			logFatal("Failed to create exFAT filesystem: %v", err)
		}
	case "ntfs":
		if err := MakeNTFS(primaryPartition); err != nil {
			logFatal("Failed to create NTFS filesystem: %v", err)
		}
	case "fat32":
		if err := MakeFAT32(primaryPartition); err != nil {
			logFatal("Failed to create FAT32 filesystem: %v", err)
		}
	}

	// Step 5: Unpack Windows ISO contents to primary partition
	func() {
		currentPhase++
		logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Extracting ISO to sources partition")
		mountPoint, err := os.MkdirTemp(os.TempDir(), "glassusb-")
		if err != nil {
			logFatal("Failed to create mount point: %v", err)
		}
		defer os.Remove(mountPoint)
		if err := MountPartition(primaryPartition, mountPoint); err != nil {
			logFatal("Failed to mount partition: %v", err)
		}
		defer func() {
			if err := UnmountPartition(mountPoint); err != nil {
				logWarn("Failed to unmount partition: %v", err)
			}
		}()
		if err := ExtractISOToLocation(iso, mountPoint); err != nil {
			logFatal("Failed to extract ISO contents: %v", err)
		}
	}()

	// Step 6: Validate Windows ISO contents on primary partition
	func() {
		if *skipValidationFlag {
			return
		}
		currentPhase++
		logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Validating ISO contents on sources partition")
		mountPoint, err := os.MkdirTemp(os.TempDir(), "glassusb-")
		if err != nil {
			logFatal("Failed to create mount point: %v", err)
		}
		defer os.Remove(mountPoint)
		if err := MountPartition(primaryPartition, mountPoint); err != nil {
			logFatal("Failed to mount partition: %v", err)
		}
		defer func() {
			if err := UnmountPartition(mountPoint); err != nil {
				logWarn("Failed to unmount partition: %v", err)
			}
		}()
		if err := ValidateISOAgainstLocation(iso, mountPoint); err != nil {
			logFatal("Failed to validate ISO contents: %v", err)
		}
	}()

	// Step 7: Write MBR to device for boot using `ms-sys`
	if gptFlag == nil || !*gptFlag {
		currentPhase++
		logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Writing MBR bootloader")
		if err := WriteMBRToPartition(primaryPartition); err != nil {
			logFatal("Failed to write MBR bootloader: %v", err)
		}
		if err := WriteMBRToPartition(blockDevice); err != nil {
			logFatal("Failed to write MBR bootloader: %v", err)
		}
	}

	// If dialog, complete it
	if dlg != nil {
		err = dlg.Complete()
		if err != nil {
			log.Panicf("Failed to complete progress dialog: %v", err)
		}
		<-dlg.Done()
	}
}
