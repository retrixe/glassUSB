package main

import (
	"flag"
	"log"
	"os"

	_ "embed"
)

const version = "1.0.0-dev"

var vFlag = flag.Bool("v", false, "")
var versionFlag = flag.Bool("version", false, "Show version")

var flashFlagSet = flag.NewFlagSet("flash", flag.ExitOnError)
var gptFlag = flashFlagSet.Bool("gpt", false,
	"Use GPT partitioning\n"+
		"Note: Only compatible with UEFI systems i.e. PCs with Windows 8 or newer")
var primaryFsFlag = flashFlagSet.String("primary-fs", "fat32",
	"Filesystem to use. If not using FAT32, UEFI:NTFS will be installed, and all\n"+
		"ISO files will be stored on a single partition.\n"+
		"Options: fat32, exfat, ntfs")
var secondaryFsFlag = flashFlagSet.String("secondary-fs", "exfat",
	"Filesystem to use for second partition if primary-fs=fat32 and ISO > 4GB\n"+
		"Options: exfat, ntfs")
var disableValidationFlag = flashFlagSet.Bool("disable-validation", false,
	"Disable validation of written files")

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
		flashFlagSet.Parse(os.Args[2:])
		args := flashFlagSet.Args()
		if len(args) != 2 {
			flashFlagSet.Usage()
			os.Exit(1)
		}

		return
	} else {
		flag.Usage()
		os.Exit(1)
	}
}
