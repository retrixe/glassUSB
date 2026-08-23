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

func GetMsSysAsProgram() (*os.File, func(), error) {
	f, err := os.CreateTemp(os.TempDir(), "ms-sys-*.bin")
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	if _, err := f.Write(MS_SYS_BIN); err != nil {
		return nil, nil, err
	}

	if err := f.Chmod(0755); err != nil {
		return nil, nil, err
	}

	return f, func() { os.Remove(f.Name()) }, nil
}

func WriteMBRToDisk(device string) error {
	msSys, cleanup, err := GetMsSysAsProgram()
	if err != nil {
		return err
	}
	defer cleanup()
	// Rufus MBR (-r) has been tested to work well and supports unattended installs.
	// By default, one would typically use the `-w` flag to automatically select a Windows-style MBR.
	if out, err := exec.Command(msSys.Name(), "-r", device).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write disk MBR to %s: %w\noutput: %s", device, err, out)
	}
	return nil
}

func WriteVBRToPartition(partition string) error {
	msSys, cleanup, err := GetMsSysAsProgram()
	if err != nil {
		return err
	}
	defer cleanup()
	if out, err := exec.Command(msSys.Name(), "-w", partition).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write VBR to %s: %w\noutput: %s", partition, err, out)
	}
	return nil
}
