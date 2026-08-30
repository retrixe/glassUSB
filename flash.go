package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/retrixe/imprint/imaging"
)

func parseFlashFlagSet(wizard bool) (bool, string, error) {
	// Look for prerequisites on system and change fs flag defaults accordingly
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
		log.Println("Invalid value provided for `-fs` flag!")
		flashFlagSet.Usage()
		os.Exit(1)
	} else if *fsFlag == "" {
		return false, "", fmt.Errorf("this system does not have any filesystem drivers supported by glassUSB, exiting...")
	} else if !slices.Contains(supportedFilesystems, *fsFlag) {
		return false, "", fmt.Errorf("this system does not have drivers for the specified filesystem (%s), exiting...", *fsFlag)
	}
	debugBypassChecksEnv := os.Getenv("__GLASSUSB_DEBUG_BYPASS_CHECKS")
	debugBypassChecks := debugBypassChecksEnv == "true" || debugBypassChecksEnv == "1"

	// Check for root permissions before proceeding
	if os.Getuid() > 0 && !debugBypassChecks {
		return false, "", fmt.Errorf("glassUSB must be run with root permissions (`sudo`) to write to devices, exiting...")
	}

	// Warn about exFAT and FAT32 limitations
	warning := ""
	addendum := "If you encounter any issues, try installing NTFS drivers on your system (Paragon NTFS for macOS, ntfs-3g for Linux)."
	if fullySupportedFsAvailable {
		addendum = "If you encounter any issues, try using NTFS instead."
	}
	switch *fsFlag {
	case "exfat":
		warning = fmt.Sprintf("%s %s", "Warning: Drives formatted with exFAT (--fs=exfat) will not boot on PCs with Secure Boot enabled.", addendum)
	case "fat32":
		warning = fmt.Sprintf("%s %s", "Warning: Using FAT32 (--fs=fat32) may cause flashing to fail for ISOs larger than 4 GB in size.", addendum)
	}
	return debugBypassChecks, warning, nil
}

func WriteWindowsISOToBlockDevice(
	ctx context.Context,
	isoFile string, blockDevice string,
	gpt bool, fs string, skipValidation bool, debugBypassChecks bool,
	logError func(string, ...any) error,
	logProgress func(string),
	logProgressRawStdout func(string),
	logProgressDisplayOnly func(string),
	logWarn func(string, ...any),
) error {
	totalPhasesNum := 7
	if gpt {
		totalPhasesNum-- // Skip MBR writing phase
	}
	if skipValidation {
		totalPhasesNum-- // Skip validation phase
	}
	if fs == "fat32" {
		totalPhasesNum-- // Skip UEFI:NTFS writing phase
	}
	totalPhases := strconv.Itoa(totalPhasesNum)
	currentPhase := 0

	// Step 1: Read ISO
	currentPhase++
	logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Reading ISO")
	file, err := os.Open(isoFile)
	if err != nil {
		return logError("failed to open ISO: %w", err)
	}
	defer file.Close()
	srcStat, err := file.Stat()
	if err != nil {
		return logError("failed to stat ISO file: %w", err)
	}
	iso, err := OpenWindowsISO(file)
	if err != nil {
		return logError("failed to read UDF filesystem on ISO: %w", err)
	}
	//totalSize := GetISOContentSize(iso)
	//log.Println("Total ISO size:", strconv.Itoa(int(totalSize)), "bytes",
	//	"("+imaging.BytesToString(int(totalSize), false)+", "+imaging.BytesToString(int(totalSize), true)+")")
	if ctx.Err() != nil {
		return logError("operation cancelled")
	}

	// Step 2: Open the block device and create a new partition table
	currentPhase++
	logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Partitioning destination drive")
	destStat, err := os.Stat(blockDevice)
	if err != nil {
		return logError("failed to get info about destination: %w", err)
	} else if destStat.Mode().Type()&os.ModeDevice == 0 {
		if !debugBypassChecks {
			return logError("destination %s is not a valid block device!", blockDevice)
		}
	}
	blockDeviceSize, err := GetBlockDeviceSize(blockDevice)
	const deviceSizeMargin = 4 * 1024 * 1024 // Extra 4 MB margin for partition table, UEFI:NTFS, etc
	if err != nil {
		return logError("failed to get size of destination: %w", err)
	} else if srcStat.Size()+deviceSizeMargin > blockDeviceSize {
		if !debugBypassChecks {
			return logError("cannot write ISO to destination: ISO size (%s) is larger than device size (%s)!",
				imaging.BytesToString(int(srcStat.Size()), true),
				imaging.BytesToString(int(blockDeviceSize), true))
		}
	}
	err = imaging.UnmountDevice(blockDevice)
	if err != nil && err != imaging.ErrNotBlockDevice { // Ignore non-block-device error here
		return logError("failed to unmount destination device: %w", err)
	}
	if ctx.Err() != nil {
		return logError("operation cancelled")
	}
	if fs == "fat32" {
		err = FormatDiskForSinglePartition(blockDevice, gpt)
	} else {
		err = FormatDiskForUEFINTFS(blockDevice, gpt)
	}
	if err != nil {
		return logError("failed to format disk: %w", err)
	}
	if ctx.Err() != nil {
		return logError("operation cancelled")
	}

	// Step 3: Write UEFI:NTFS to second partition
	if fs != "fat32" {
		currentPhase++
		logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Writing UEFI:NTFS bootloader")
		err = WriteUEFINTFSToPartition(blockDevice, 2)
		if err != nil {
			return logError("failed to write UEFI bootloader to second partition: %w", err)
		}
		if ctx.Err() != nil {
			return logError("operation cancelled")
		}
	}

	// Step 4: Format primary partition depending on fs flag
	currentPhase++
	logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Creating sources partition")
	primaryPartition := GetBlockDevicePartition(blockDevice, 1)
	windowsVolumeLabel := iso.GetLogicalVolumeIdentifier()
	if windowsVolumeLabel == "" {
		windowsVolumeLabel = "Windows USB"
	}
	switch fs {
	case "exfat":
		if err := MakeExFAT(primaryPartition, sanitizeExFATLabel(windowsVolumeLabel)); err != nil {
			return logError("failed to create exFAT filesystem: %w", err)
		}
	case "ntfs":
		if err := MakeNTFS(primaryPartition, sanitizeNTFSLabel(windowsVolumeLabel)); err != nil {
			return logError("failed to create NTFS filesystem: %w", err)
		}
	case "fat32":
		if err := MakeFAT32(primaryPartition, sanitizeFATLabel(windowsVolumeLabel)); err != nil {
			return logError("failed to create FAT32 filesystem: %w", err)
		}
	}
	if ctx.Err() != nil {
		return logError("operation cancelled")
	}

	// Step 5: Unpack Windows ISO contents to primary partition
	if err = func() error {
		currentPhase++
		progStr := "Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Extracting ISO to sources partition"
		logProgress(progStr)
		mountPoint, err := os.MkdirTemp(os.TempDir(), "glassusb-")
		if err != nil {
			return logError("failed to create mount point: %w", err)
		}
		defer os.Remove(mountPoint)
		if err := MountPartition(primaryPartition, mountPoint); err != nil {
			return logError("failed to mount partition: %w", err)
		}
		defer func() {
			if err := UnmountPartition(mountPoint); err != nil {
				logWarn("Failed to unmount partition: %v", err)
			}
		}()
		if ctx.Err() != nil {
			return logError("operation cancelled")
		}
		logFn := func(log string) {
			logProgressRawStdout(log)
			logProgressDisplayOnly(progStr + "\n" + strings.TrimSpace(log))
		}
		if err := ExtractISOToLocation(ctx, logFn, iso, mountPoint); err != nil {
			return logError("failed to extract ISO contents: %w", err)
		}
		return nil
	}(); err != nil {
		return err
	}

	// Step 6: Validate Windows ISO contents on primary partition
	if err = func() error {
		if skipValidation {
			return nil
		}
		currentPhase++
		progStr := "Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Validating ISO contents on sources partition"
		logProgress(progStr)
		mountPoint, err := os.MkdirTemp(os.TempDir(), "glassusb-")
		if err != nil {
			return logError("failed to create mount point: %w", err)
		}
		defer os.Remove(mountPoint)
		if err := MountPartition(primaryPartition, mountPoint); err != nil {
			return logError("failed to mount partition: %w", err)
		}
		defer func() {
			if err := UnmountPartition(mountPoint); err != nil {
				logWarn("Failed to unmount partition: %v", err)
			}
		}()
		if ctx.Err() != nil {
			return logError("operation cancelled")
		}
		logFn := func(log string) {
			logProgressRawStdout(log)
			logProgressDisplayOnly(progStr + "\n" + strings.TrimSpace(log))
		}
		if err := ValidateISOAgainstLocation(ctx, logFn, iso, mountPoint); err != nil {
			return logError("failed to validate ISO contents: %w", err)
		}
		return nil
	}(); err != nil {
		return err
	}

	// Step 7: Write MBR to device for boot using `ms-sys`
	if !gpt {
		currentPhase++
		logProgress("Phase " + strconv.Itoa(currentPhase) + "/" + totalPhases + ": Writing MBR bootloader")
		if err := WriteVBRToPartition(primaryPartition); err != nil {
			return logError("failed to write VBR bootloader: %w", err)
		}
		if err := WriteMBRToDisk(blockDevice); err != nil {
			return logError("failed to write MBR bootloader: %w", err)
		}
	}

	return nil
}
