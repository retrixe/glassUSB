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

func WriteMBRToPartition(partition string) error {
	msSys, err := GetMsSysAsProgram()
	if err != nil {
		return err
	}
	if out, err := exec.Command(msSys.Name(), "-w", partition).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write MBR bootloader to %s: %w\noutput: %s", partition, err, out)
	}
	return nil
}
