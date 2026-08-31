package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	_ "embed"

	"github.com/ncruces/zenity"
	"github.com/retrixe/imprint/imaging"
)

const version = "1.0.0-alpha.2"

var vFlag = flag.Bool("v", false, "")
var versionFlag = flag.Bool("version", false, "Show version")

func mainUsage() {
	println("Usage: glassUSB [command] [options]")
	println("\nglassUSB is a simple tool for Linux systems to create a bootable Windows USB")
	println("installer from a Windows ISO / DVD image. It provides both an easy to use CLI and")
	println("interactive GUI wizard.")
	println("\nNote: glassUSB should be run with root permissions (`sudo`) to write to devices.")
	println("The wizard must be run with `sudo -E` on Linux to allow dialogs to show correctly.")
	println("\nAvailable commands:")
	println("  flash       Flash a Windows ISO to a specific USB device.")
	println("  wizard      (Beta) Start a GUI wizard for flashing Windows ISOs to a USB device.")
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
	println("\nNote: The wizard must be run with `sudo -E` on Linux, to allow Zenity to show")
	println("dialogs correctly.")
	println("\nOptions:")
	flashFlagSet.PrintDefaults()
}

/*
	 TODO: Support FAT32 + secondary-fs or some kind of splitting
		var secondaryFsFlag = flashFlagSet.String("secondary-fs", "exfat",
			"Filesystem to use for second partition if primary-fs=fat32 and ISO > 4GB\n"+
				"Options: exfat, ntfs")
*/

//go:embed binaries/uefi/uefi-ntfs.img
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
		if err := flashCommand(); err != nil {
			os.Exit(1)
		}
	} else if len(os.Args) >= 2 && os.Args[1] == "wizard" {
		flashFlagSet.Usage = flashWizardUsage
		if err := wizardCommand(); err != nil {
			log.Fatalln(err)
		}
	} else {
		flag.Usage()
		os.Exit(1)
	}
}

func flashCommand() error {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	log.SetPrefix("[glassUSB] ")

	// Parse flags
	debugBypassChecks, warning, err := parseFlashFlagSet(false)
	if err != nil {
		log.Println(err)
		return err
	} else if warning != "" {
		log.Println(warning)
	}
	args := flashFlagSet.Args()

	log.Println("Selected ISO:", args[0])
	log.Println("Target device path:", args[1])

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt,
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer cancel()

	// Write Windows ISO to block device
	logWarn := log.Printf
	logProgress := func(message string) { log.Println(message) }
	logProgressRawStdout := func(log string) { print(log) }
	logProgressDisplayOnly := func(log string) {}
	err = WriteWindowsISOToBlockDevice(
		ctx,
		args[0], args[1],
		*gptFlag, *fsFlag, *skipValidationFlag, debugBypassChecks,
		logProgress, logProgressRawStdout, logProgressDisplayOnly, logWarn,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	log.Println("The flash process completed successfully! You can now boot from this USB to install Windows.")
	return nil
}

func wizardCommand() error {
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
	displayWarning := func(warning string) {
		zenity.Warning(warning,
			zenity.Width(640),
			zenity.WindowIcon(zenity.WarningIcon),
			zenity.Title("glassUSB Media Creation Wizard"),
			zenity.Icon(zenity.WarningIcon),
			zenity.OKLabel("Continue"))
	}
	logWarn := func(format string, v ...any) {
		log.Printf(format, v...)
		displayWarning(fmt.Sprintf(format, v...))
	}
	displayError := func(err error) {
		zenity.Error(imaging.CapitalizeString(err.Error()),
			zenity.Width(640),
			zenity.WindowIcon(zenity.ErrorIcon),
			zenity.Title("glassUSB Media Creation Wizard"),
			zenity.Icon(zenity.ErrorIcon),
			zenity.OKLabel("Exit"))
	}
	logProgressRawStdout := func(log string) { print(log) }
	logProgressDisplayOnly := func(log string) {
		if dlg != nil {
			if runtime.GOOS == "linux" {
				// Hack: Replace newlines with literal \n for zenity on Linux, which seems to not handle newlines in progress dialog text properly
				// Meanwhile, macOS doesn't even show newlines lol
				log = strings.ReplaceAll(log, "\n", "\\n")
			}
			dlg.Text(log)
		}
	}

	// Parse flags
	debugBypassChecks, warning, err := parseFlashFlagSet(true)
	if err != nil {
		displayError(err)
		return err
	} else if warning != "" {
		log.Println(warning)
		displayWarning(warning)
	}
	args := flashFlagSet.Args()

	// Prompt user for ISO and device
	err = zenity.Question(`This wizard will guide you through the process of creating a Windows installation USB drive.

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
		return fmt.Errorf("failed to continue with wizard: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("failed to open file dialog: %w", err)
		displayError(err)
		return err
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
		return fmt.Errorf("failed to continue with wizard: %w", err)
	}

	var device, deviceName string
	for {
		devices, err := imaging.GetDevices(imaging.SystemPlatform)
		if err != nil {
			err = fmt.Errorf("failed to get connected drives: %w", err)
			displayError(err)
			return err
		} else if len(devices) == 0 {
			if runtime.GOOS == "linux" { // No extra button on Windows/macOS
				err = zenity.Error("Failed to find any USB devices connected to your computer.\n\n"+
					"Please connect a USB flash drive and try again.",
					zenity.Width(640),
					zenity.WindowIcon(zenity.ErrorIcon),
					zenity.Title("glassUSB - Select target USB drive"),
					zenity.Icon(zenity.ErrorIcon),
					zenity.OKLabel("Exit"),
					zenity.ExtraButton("Rescan devices"))
			} else {
				err = zenity.Error("Failed to find any USB devices connected to your computer.\n\n"+
					"Please connect a USB flash drive and try again.",
					zenity.Width(640),
					zenity.WindowIcon(zenity.ErrorIcon),
					zenity.Title("glassUSB - Select target USB drive"),
					zenity.Icon(zenity.ErrorIcon),
					zenity.OKLabel("Exit"))
			}
			if err == nil {
				return fmt.Errorf("no USB devices connected, exiting...")
			} else if !errors.Is(err, zenity.ErrExtraButton) {
				return fmt.Errorf("failed to continue with wizard: %w", err)
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
		if runtime.GOOS == "linux" { // No extra button on Windows/macOS
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
		} else {
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
			)
		}
		if errors.Is(err, zenity.ErrExtraButton) {
			continue
		} else if err != nil {
			return fmt.Errorf("failed to continue with wizard: %w", err)
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
		return fmt.Errorf("failed to continue with wizard: %w", err)
	}

	dlg, err = zenity.Progress(
		zenity.Width(640),
		zenity.WindowIcon(zenity.InfoIcon),
		zenity.Title("glassUSB Media Creation Wizard"),
		zenity.Icon(zenity.NoIcon),
		zenity.Pulsate(),
		// TODO: Could we use TimeRemaining at the flash stage
		zenity.CancelLabel("Cancel"),
		zenity.OKLabel("Finish"))
	if err != nil {
		return fmt.Errorf("failed to continue with wizard: %w", err)
	}
	defer dlg.Close()

	args = []string{isoPath, deviceName}
	log.Println("Selected ISO:", isoPath)
	log.Println("Target device:", device)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt,
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer cancel()
	go func() {
		if dlg != nil {
			<-dlg.Done()
			cancel()
		}
	}()

	// Write Windows ISO to block device
	err = WriteWindowsISOToBlockDevice(
		ctx,
		args[0], args[1],
		*gptFlag, *fsFlag, *skipValidationFlag, debugBypassChecks,
		logProgress, logProgressRawStdout, logProgressDisplayOnly, logWarn,
	)
	if err != nil {
		displayError(err)
		return err
	}

	signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	logProgress("The flash process completed successfully! You can now boot from this USB to install Windows.")

	// Complete the dialog
	err = dlg.Complete()
	if err != nil {
		return fmt.Errorf("failed to complete progress dialog: %w", err)
	}
	<-ctx.Done() // The context will be cancelled when dlg.Done() by the goroutine fired earlier

	return nil
}
