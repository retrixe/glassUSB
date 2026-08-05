package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
)

//go:generate sh build-ms-sys.sh

//go:embed binaries/ms-sys
var MS_SYS_BIN []byte

func GetMsSysAsProgram() (*os.File, error) {
	f, err := os.CreateTemp(os.TempDir(), "ms-sys-*.bin")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Write(MS_SYS_BIN); err != nil {
		return nil, err
	}

	if err := f.Chmod(0755); err != nil {
		return nil, err
	}

	return f, nil
}

func WriteMBRToDisk(device string) error {
	msSys, err := GetMsSysAsProgram()
	if err != nil {
		return err
	}
	// FIXME: Test if CSM works at all, before experimenting with Rufus MBR (-r)
	// Switch to Rufus MBR (-r) in the future, as it supports unattended installs
	// First, we need to test if CSM works at all, before experimenting with Rufus MBR
	if out, err := exec.Command(msSys.Name(), "-r", device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write disk MBR to %s: %w\noutput: %s", device, err, out)
	}
	return nil
}

func WriteVBRToPartition(partition string) error {
	msSys, err := GetMsSysAsProgram()
	if err != nil {
		return err
	}
	if out, err := exec.Command(msSys.Name(), "-w", partition).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write VBR to %s: %w\noutput: %s", partition, err, out)
	}
	return nil
}
